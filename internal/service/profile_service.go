package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/config"
	"github.com/unkmonster/tmd/internal/database"
	"github.com/unkmonster/tmd/internal/downloader"
	"github.com/unkmonster/tmd/internal/naming"
	"github.com/unkmonster/tmd/internal/path"
	"github.com/unkmonster/tmd/internal/utils"
)

const maxProfileDownloadRoutine = 20

// indexedRequest 带索引的请求（内部使用）
type indexedRequest struct {
	index   int
	request ProfileRequest
}

// ProfileService Profile 下载服务
type ProfileService struct {
	client            *resty.Client
	additionalClients []*resty.Client
	db                *sqlx.DB
	conf              *config.Config
	appRootPath       string
}

// NewProfileService 创建 Profile 服务
func NewProfileService(client *resty.Client, additionalClients []*resty.Client, db *sqlx.DB, conf *config.Config, appRootPath string) *ProfileService {
	return &ProfileService{
		client:            client,
		additionalClients: additionalClients,
		db:                db,
		conf:              conf,
		appRootPath:       appRootPath,
	}
}

// ProfileRequest 单个 Profile 请求（用于内部处理）
type ProfileRequest struct {
	ScreenName  string
	UserTitle   string
	Name        string
	UserID      uint64
	AvatarURL   string
	BannerURL   string
	Description string
	Location    string
	URL         string
	Verified    bool
	Protected   bool
	CreatedAt   string
}

// ExecuteDownloadProfiles 执行 Profile 下载
func (s *ProfileService) ExecuteDownloadProfiles(ctx context.Context, req *DownloadProfilesRequest) error {
	pathHelper, err := path.NewStorePath(s.conf.RootPath)
	if err != nil {
		return fmt.Errorf("failed to create store path: %w", err)
	}

	versionManager := downloader.NewVersionManagerWithWriter(pathHelper.Data, nil)
	fileWriter := downloader.NewFileWriter(versionManager)
	versionManager.SetFileWriter(fileWriter)
	dwn := downloader.NewDownloader(fileWriter)

	storage, err := newProfileStorageManager(pathHelper.Users)
	if err != nil {
		return fmt.Errorf("failed to create profile storage: %w", err)
	}
	storage.setVersionManager(versionManager)

	clients := make([]*resty.Client, 0)
	clients = append(clients, s.client)
	clients = append(clients, s.additionalClients...)
	fetcher := newTwitterFetcherWithClients(clients)

	requests := make([]ProfileRequest, 0, len(req.Users))
	for _, user := range req.Users {
		requests = append(requests, ProfileRequest{
			ScreenName: user.ScreenName,
			UserTitle:  user.Title(),
			Name:       user.Name,
			UserID:     user.Id,
			AvatarURL:  user.AvatarURL,
			BannerURL:  user.BannerURL,
		})
	}

	if len(requests) == 0 {
		log.Infoln("No profile requests to download")
		return nil
	}

	seen := make(map[string]bool)
	uniqueRequests := make([]ProfileRequest, 0)
	for _, req := range requests {
		if !seen[req.ScreenName] {
			seen[req.ScreenName] = true
			uniqueRequests = append(uniqueRequests, req)
		}
	}

	log.Infoln("Starting profile download for", len(uniqueRequests), "users")
	results := s.downloadMultiple(ctx, uniqueRequests, storage, fetcher, dwn, fileWriter)

	success := 0
	failed := 0
	skipped := 0
	for _, r := range results {
		if r.Success {
			success++
		} else if r.Error != nil {
			failed++
		} else {
			skipped++
		}
	}

	log.Infoln("profile download completed - total:", len(results), "success:", success, "failed:", failed, "skipped:", skipped)

	return nil
}

func (s *ProfileService) downloadSingle(
	ctx context.Context,
	req ProfileRequest,
	storage *profileStorageManager,
	fetcher *twitterFetcher,
	dwn downloader.Downloader,
	fileWriter downloader.FileWriter,
) *ProfileDownloadResult {
	startTime := time.Now()
	result := &ProfileDownloadResult{
		ScreenName: req.ScreenName,
		Files:      make([]FileResult, 0),
	}

	config := DefaultProfileServiceConfig()

	var profile *ProfileInfo
	var err error

	needAPICall := req.UserID == 0 || req.AvatarURL == ""

	if needAPICall && fetcher != nil {
		log.Debugln("calling API to fetch profile data for:", req.ScreenName)
		profile, err = fetcher.FetchProfile(ctx, req.ScreenName)
		if err != nil {
			result.Error = fmt.Errorf("failed to fetch profile: %w", err)
			if s.db != nil {
				database.MarkUserInaccessible(s.db, 0, req.ScreenName)
			}
			return result
		}
	} else {
		profile = &ProfileInfo{
			ID:          req.UserID,
			Name:        req.Name,
			ScreenName:  req.ScreenName,
			AvatarURL:   req.AvatarURL,
			BannerURL:   req.BannerURL,
			Description: req.Description,
			Location:    req.Location,
			URL:         req.URL,
			Verified:    req.Verified,
			Protected:   req.Protected,
			CreatedAt:   req.CreatedAt,
		}
		log.Debugln("using pre-fetched profile data, no API call needed")
	}

	result.Profile = profile
	log.Debugln("profile fetched:", profile.Name, "(id:", profile.ID, ")")

	userTitle := req.UserTitle
	if userTitle == "" {
		userTitle = fmt.Sprintf("%s(%s)", profile.Name, req.ScreenName)
	}
	userTitle = utils.WinFileNameWithMaxLen(userTitle, naming.MaxFileNameLen)

	var userDir string

	if s.db != nil && profile.ID != 0 {
		userDir, err = s.syncUserDirectory(profile, userTitle, req.ScreenName, storage)
		if err != nil {
			result.Error = fmt.Errorf("failed to sync directory: %w", err)
			return result
		}
	} else {
		userDir, err = storage.ensureDirectory(userTitle)
		if err != nil {
			result.Error = fmt.Errorf("failed to create directory: %w", err)
			return result
		}
	}

	log.Debugln("directory ready:", userDir)

	fetchedAt := time.Now()

	if profile.AvatarURL != "" {
		avatarResult := s.downloadAvatar(ctx, userTitle, req.ScreenName, profile.AvatarURL, fetchedAt, storage, fetcher, dwn, fileWriter, config)
		result.Files = append(result.Files, avatarResult)
	}

	if profile.BannerURL != "" {
		bannerResult := s.downloadFile(ctx, userTitle, req.ScreenName, FileTypeBanner, profile.BannerURL, ".jpg", fetchedAt, "banner", storage, fetcher, dwn, fileWriter, config)
		result.Files = append(result.Files, bannerResult)
	}

	descResult := s.saveContent(userTitle, FileTypeDescription, []byte(profile.Description), fetchedAt, storage, fileWriter, config)
	result.Files = append(result.Files, descResult)

	profileResult := s.saveProfileJSON(userTitle, req.ScreenName, profile, fetchedAt, storage, fileWriter, config)
	result.Files = append(result.Files, profileResult)

	result.Success = true
	for _, file := range result.Files {
		if file.Status == StatusFailed {
			result.Success = false
			break
		}
	}

	result.DownloadTime = time.Since(startTime)

	return result
}

func (s *ProfileService) syncUserDirectory(profile *ProfileInfo, userTitle, screenName string, storage *profileStorageManager) (string, error) {
	if err := database.SyncUser(s.db, profile.ID, profile.Name, screenName, profile.Protected, 0, true); err != nil {
		return "", err
	}

	entity, err := database.LocateUserEntity(s.db, profile.ID, storage.usersBasePath)
	if err != nil {
		return "", err
	}

	expectedTitle := userTitle

	if entity == nil {
		entity = &database.UserEntity{
			Uid:       profile.ID,
			ParentDir: storage.usersBasePath,
			Name:      expectedTitle,
		}
		userDir := filepath.Join(storage.usersBasePath, expectedTitle)
		if err := os.MkdirAll(userDir, 0755); err != nil {
			return "", err
		}
		if err := database.CreateUserEntity(s.db, entity); err != nil {
			return "", err
		}
		log.Infoln("new user directory created:", userDir)
		return ensureProfileDirs(userDir)
	}

	oldUserDir, err := entity.Path()
	if err != nil {
		return "", err
	}
	if entity.Name == expectedTitle {
		if err := os.MkdirAll(oldUserDir, 0755); err != nil && !os.IsExist(err) {
			return "", err
		}
		return ensureProfileDirs(oldUserDir)
	}

	newUserDir := filepath.Join(storage.usersBasePath, expectedTitle)
	if err := os.Rename(oldUserDir, newUserDir); err != nil {
		if os.IsNotExist(err) {
			if mkdirErr := os.MkdirAll(newUserDir, 0755); mkdirErr != nil {
				return "", mkdirErr
			}
		} else {
			return "", err
		}
	}

	entity.Name = expectedTitle
	if err := database.UpdateUserEntity(s.db, entity); err != nil {
		return "", err
	}

	log.Infoln("user directory renamed:", oldUserDir, "->", newUserDir)
	return ensureProfileDirs(newUserDir)
}

func (s *ProfileService) downloadMultiple(
	ctx context.Context,
	requests []ProfileRequest,
	storage *profileStorageManager,
	fetcher *twitterFetcher,
	dwn downloader.Downloader,
	fileWriter downloader.FileWriter,
) []*ProfileDownloadResult {
	if len(requests) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	results := make([]*ProfileDownloadResult, len(requests))
	var wg sync.WaitGroup
	var mu sync.Mutex

	numRoutine := min(len(requests), maxProfileDownloadRoutine)

	reqChan := make(chan indexedRequest, len(requests))
	for i, req := range requests {
		reqChan <- indexedRequest{index: i, request: req}
	}
	close(reqChan)

	for i := 0; i < numRoutine; i++ {
		wg.Add(1)
		go s.profileDownloaderWorker(ctx, cancel, &wg, &mu, results, reqChan, storage, fetcher, dwn, fileWriter)
	}

	wg.Wait()
	return results
}

func (s *ProfileService) profileDownloaderWorker(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	wg *sync.WaitGroup,
	mu *sync.Mutex,
	results []*ProfileDownloadResult,
	reqChan <-chan indexedRequest,
	storage *profileStorageManager,
	fetcher *twitterFetcher,
	dwn downloader.Downloader,
	fileWriter downloader.FileWriter,
) {
	defer wg.Done()
	defer func() {
		if p := recover(); p != nil {
			log.Errorf("[profileDownloaderWorker] panic recovered: %v", p)
			cancel(fmt.Errorf("panic: %v", p))

			for ir := range reqChan {
				mu.Lock()
				results[ir.index] = &ProfileDownloadResult{
					ScreenName: ir.request.ScreenName,
					Success:    false,
					Error:      fmt.Errorf("download cancelled due to panic"),
				}
				mu.Unlock()
			}
		}
	}()

	for {
		select {
		case ir, ok := <-reqChan:
			if !ok {
				return
			}
			result := s.downloadSingle(ctx, ir.request, storage, fetcher, dwn, fileWriter)
			if result.Error != nil {
				log.Errorln("profile download failed:", ir.request.ScreenName, "-", result.Error)

				if errors.Is(result.Error, syscall.ENOSPC) {
					cancel(result.Error)
					for remainingIr := range reqChan {
						mu.Lock()
						results[remainingIr.index] = &ProfileDownloadResult{
							ScreenName: remainingIr.request.ScreenName,
							Success:    false,
							Error:      fmt.Errorf("download cancelled: disk full"),
						}
						mu.Unlock()
					}
					return
				}
			}

			mu.Lock()
			results[ir.index] = result
			mu.Unlock()

		case <-ctx.Done():
			for ir := range reqChan {
				mu.Lock()
				results[ir.index] = &ProfileDownloadResult{
					ScreenName: ir.request.ScreenName,
					Success:    false,
					Error:      ctx.Err(),
				}
				mu.Unlock()
			}
			return
		}
	}
}

func (s *ProfileService) downloadAvatar(
	ctx context.Context,
	userTitle, screenName, url string,
	fetchedAt time.Time,
	storage *profileStorageManager,
	fetcher *twitterFetcher,
	dwn downloader.Downloader,
	fileWriter downloader.FileWriter,
	config *ProfileServiceConfig,
) FileResult {
	ext := downloader.ExtractImageExtFromURL(url)
	return s.downloadFile(ctx, userTitle, screenName, FileTypeAvatar,
		getHighResAvatarURL(url, config.AvatarQuality), ext, fetchedAt, "avatar",
		storage, fetcher, dwn, fileWriter, config)
}

func (s *ProfileService) downloadFile(
	ctx context.Context,
	userTitle, screenName string,
	fileType FileType,
	url, defaultExt string,
	fetchedAt time.Time,
	label string,
	storage *profileStorageManager,
	fetcher *twitterFetcher,
	dwn downloader.Downloader,
	_ downloader.FileWriter,
	config *ProfileServiceConfig,
) FileResult {
	filePath := storage.getFilePathWithExt(userTitle, fileType, defaultExt)
	client := fetcher.Client()

	downloadReq := downloader.DownloadRequest{
		Context:     ctx,
		Client:      client,
		URL:         url,
		Destination: filePath,
		Options: downloader.DownloadOptions{
			SkipUnchanged: config.SkipUnchanged,
			CreateVersion: config.EnableVersioning,
			SetModTime:    &fetchedAt,
		},
	}

	result, err := dwn.Download(downloadReq)
	if err != nil {
		log.Errorln(label+" download failed:", screenName, "-", err)
		return FileResult{FileType: fileType, Status: StatusFailed, Error: err}
	}

	status := StatusDownloaded
	if result.Skipped {
		status = StatusSkipped
	}
	return FileResult{FileType: fileType, Status: status, FilePath: result.FilePath, OldSize: result.OldSize, NewSize: result.FileSize}
}

func (s *ProfileService) saveProfileJSON(
	userTitle, screenName string,
	profile *ProfileInfo,
	fetchedAt time.Time,
	storage *profileStorageManager,
	fileWriter downloader.FileWriter,
	config *ProfileServiceConfig,
) FileResult {
	data, err := profileToJSON(profile)
	if err != nil {
		log.Errorln("profile JSON serialize failed:", screenName, "-", err)
		return FileResult{FileType: FileTypeProfile, Status: StatusFailed, Error: err}
	}
	return s.saveContent(userTitle, FileTypeProfile, data, fetchedAt, storage, fileWriter, config)
}

func (s *ProfileService) saveContent(
	userTitle string,
	fileType FileType,
	data []byte,
	fetchedAt time.Time,
	storage *profileStorageManager,
	fileWriter downloader.FileWriter,
	config *ProfileServiceConfig,
) FileResult {
	filePath := storage.getFilePath(userTitle, fileType)

	writeReq := downloader.WriteRequest{
		Path: filePath,
		Data: data,
		Options: downloader.WriteOptions{
			CreateVersion: config.EnableVersioning,
			SkipUnchanged: config.SkipUnchanged,
			ModTime:       &fetchedAt,
		},
	}

	result, err := fileWriter.Write(writeReq)
	if err != nil {
		return FileResult{FileType: fileType, FilePath: filePath, Status: StatusFailed, Error: err}
	}

	status := StatusDownloaded
	if result.Skipped {
		status = StatusSkipped
	}
	return FileResult{FileType: fileType, Status: status, FilePath: filePath, OldSize: result.OldSize, NewSize: result.NewSize}
}
