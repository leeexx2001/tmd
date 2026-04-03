package profile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/unkmonster/tmd/internal/database"
	"github.com/unkmonster/tmd/internal/utils"
)

const (
	profileDirName    = ".loongtweet"
	profileSubDirName = ".profile"
	versionsDirName   = ".versions"
)

var MaxDownloadRoutine = 20

func extractExtFromURL(url string) string {
	ext := path.Ext(url)
	ext = strings.ToLower(ext)

	validExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	if validExts[ext] {
		return ext
	}
	return ".jpg"
}

type ProfileDownloader struct {
	config  *Config
	storage *FileStorageManager
	fetcher Fetcher
	db      *sqlx.DB
}

func NewProfileDownloader(config *Config, storage *FileStorageManager, fetcher Fetcher) *ProfileDownloader {
	if config == nil {
		config = DefaultConfig()
	}

	return &ProfileDownloader{
		config:  config,
		storage: storage,
		fetcher: fetcher,
	}
}

func NewProfileDownloaderWithClients(config *Config, storage *FileStorageManager, clients []*resty.Client) *ProfileDownloader {
	if config == nil {
		config = DefaultConfig()
	}

	fetcher := NewTwitterFetcherWithClients(clients)

	return &ProfileDownloader{
		config:  config,
		storage: storage,
		fetcher: fetcher,
	}
}

func NewProfileDownloaderWithDB(config *Config, storage *FileStorageManager, clients []*resty.Client, db *sqlx.DB) *ProfileDownloader {
	if config == nil {
		config = DefaultConfig()
	}

	fetcher := NewTwitterFetcherWithClients(clients)

	return &ProfileDownloader{
		config:  config,
		storage: storage,
		fetcher: fetcher,
		db:      db,
	}
}

func (pd *ProfileDownloader) Fetcher() Fetcher {
	return pd.fetcher
}

type DownloadRequest struct {
	ScreenName  string
	UserTitle   string // 用于目录名，格式: Name(ScreenName)
	Name        string // 纯净的显示名称(仅Name，不含ScreenName)
	UserID      uint64
	AvatarURL   string // 可选，如果提供则跳过API获取
	BannerURL   string // 可选，如果提供则跳过API获取
	Description string // 用户简介
	Location    string // 位置
	URL         string // 个人链接
	Verified    bool   // 是否认证
	Protected   bool   // 是否受保护
	CreatedAt   string // 账号创建时间
}

type indexedRequest struct {
	index   int
	request DownloadRequest
}

func (pd *ProfileDownloader) Download(ctx context.Context, req DownloadRequest) (*DownloadResult, error) {
	startTime := time.Now()
	result := &DownloadResult{
		ScreenName: req.ScreenName,
		Files:      make([]FileResult, 0),
	}

	log.Infoln("downloading profile:", req.ScreenName)

	var profile *ProfileInfo
	var err error

	// 判断是否需要调用API获取用户数据
	// 如果 UserID 为 0 或 AvatarURL 为空，说明没有预获取数据，需要调用API
	needAPICall := req.UserID == 0 || req.AvatarURL == ""

	if needAPICall && pd.fetcher != nil {
		// 调用API获取用户数据
		log.Debugln("calling API to fetch profile data for:", req.ScreenName)
		profile, err = pd.fetcher.FetchProfile(ctx, req.ScreenName)
		if err != nil {
			result.Error = fmt.Errorf("failed to fetch profile: %w", err)
			return result, result.Error
		}
	} else {
		// 使用预获取的数据
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
	userTitle = utils.WinFileName(userTitle)

	var userDir string

	if pd.db != nil && profile.ID != 0 {
		userDir, err = pd.syncUserDirectory(profile, userTitle, req.ScreenName)
		if err != nil {
			result.Error = fmt.Errorf("failed to sync directory: %w", err)
			return result, result.Error
		}
	} else {
		userDir, err = pd.storage.EnsureDirectory(userTitle, req.ScreenName)
		if err != nil {
			result.Error = fmt.Errorf("failed to create directory: %w", err)
			return result, result.Error
		}
	}

	log.Debugln("directory ready:", userDir)

	fetchedAt := time.Now()

	if profile.AvatarURL != "" {
		avatarURL := GetHighResAvatarURL(profile.AvatarURL, pd.config.AvatarQuality)
		avatarResult := pd.downloadAvatar(ctx, userTitle, req.ScreenName, avatarURL, fetchedAt)
		result.Files = append(result.Files, avatarResult)
	}

	if profile.BannerURL != "" {
		bannerResult := pd.downloadBanner(ctx, userTitle, req.ScreenName, profile.BannerURL, fetchedAt)
		result.Files = append(result.Files, bannerResult)
	}

	descResult := pd.saveDescription(userTitle, req.ScreenName, profile.Description, fetchedAt)
	result.Files = append(result.Files, descResult)

	profileResult := pd.saveProfileJSON(userTitle, req.ScreenName, profile, fetchedAt)
	result.Files = append(result.Files, profileResult)

	result.Success = true
	for _, file := range result.Files {
		if file.Status == StatusFailed {
			result.Success = false
			break
		}
	}

	result.DownloadTime = time.Since(startTime)

	log.Infoln("profile done:", req.ScreenName, "-", len(result.Files), "files")

	return result, nil
}

func (pd *ProfileDownloader) syncUserDirectory(profile *ProfileInfo, userTitle, screenName string) (string, error) {
	usrdb, err := database.GetUserById(pd.db, profile.ID)
	if err != nil {
		return "", err
	}

	isNew := usrdb == nil
	renamed := false

	if isNew {
		usrdb = &database.User{}
		usrdb.Id = profile.ID
	} else {
		renamed = usrdb.Name != profile.Name || usrdb.ScreenName != screenName
	}

	usrdb.FriendsCount = 0
	usrdb.IsProtected = profile.Protected
	usrdb.Name = profile.Name
	usrdb.ScreenName = screenName

	if isNew {
		if err = database.CreateUser(pd.db, usrdb); err != nil {
			return "", err
		}
	} else {
		if err = database.UpdateUser(pd.db, usrdb); err != nil {
			return "", err
		}
	}

	if renamed || isNew {
		if err = database.RecordUserPreviousName(pd.db, profile.ID, profile.Name, screenName); err != nil {
			log.Debugln("failed to record previous name:", err)
		}
	}

	entity, err := database.LocateUserEntity(pd.db, profile.ID, pd.storage.usersBasePath)
	if err != nil {
		return "", err
	}

	expectedTitle := userTitle

	if entity == nil {
		entity = &database.UserEntity{
			Uid:       profile.ID,
			ParentDir: pd.storage.usersBasePath,
			Name:      expectedTitle,
		}
		userDir := filepath.Join(pd.storage.usersBasePath, expectedTitle)
		if err := os.MkdirAll(userDir, 0755); err != nil {
			return "", err
		}
		if err := database.CreateUserEntity(pd.db, entity); err != nil {
			return "", err
		}
		profileDir := filepath.Join(userDir, profileDirName, profileSubDirName)
		if err := os.MkdirAll(profileDir, 0755); err != nil {
			return "", err
		}
		versionsDir := filepath.Join(profileDir, versionsDirName)
		if err := os.MkdirAll(versionsDir, 0755); err != nil {
			return "", err
		}
		return profileDir, nil
	}

	oldUserDir := entity.Path()
	if entity.Name == expectedTitle {
		if err := os.MkdirAll(oldUserDir, 0755); err != nil && !os.IsExist(err) {
			return "", err
		}
		profileDir := filepath.Join(oldUserDir, profileDirName, profileSubDirName)
		if err := os.MkdirAll(profileDir, 0755); err != nil {
			return "", err
		}
		versionsDir := filepath.Join(profileDir, versionsDirName)
		if err := os.MkdirAll(versionsDir, 0755); err != nil {
			return "", err
		}
		return profileDir, nil
	}

	newUserDir := filepath.Join(pd.storage.usersBasePath, expectedTitle)
	if err := os.Rename(oldUserDir, newUserDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(newUserDir, 0755); err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	entity.Name = expectedTitle
	if err := database.UpdateUserEntity(pd.db, entity); err != nil {
		return "", err
	}

	log.Debugln("user directory renamed:", oldUserDir, "->", newUserDir)
	profileDir := filepath.Join(newUserDir, profileDirName, profileSubDirName)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return "", err
	}
	versionsDir := filepath.Join(profileDir, versionsDirName)
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return "", err
	}
	return profileDir, nil
}

func (pd *ProfileDownloader) DownloadMultiple(ctx context.Context, requests []DownloadRequest) []*DownloadResult {
	if len(requests) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil) // 确保 cancel 在所有情况下都被调用

	results := make([]*DownloadResult, len(requests))
	var wg sync.WaitGroup
	var mu sync.Mutex

	numRoutine := min(len(requests), MaxDownloadRoutine)

	reqChan := make(chan indexedRequest, len(requests))
	for i, req := range requests {
		reqChan <- indexedRequest{index: i, request: req}
	}
	close(reqChan)

	for i := 0; i < numRoutine; i++ {
		wg.Add(1)
		go pd.profileDownloader(ctx, cancel, &wg, &mu, results, reqChan)
	}

	wg.Wait()
	return results
}

func (pd *ProfileDownloader) profileDownloader(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	wg *sync.WaitGroup,
	mu *sync.Mutex,
	results []*DownloadResult,
	reqChan <-chan indexedRequest,
) {
	defer wg.Done()
	defer func() {
		if p := recover(); p != nil {
			log.Errorln("profile downloader panic:", p)
			cancel(fmt.Errorf("panic: %v", p))

			// 把 channel 中剩余的任务标记为失败
			for ir := range reqChan {
				mu.Lock()
				results[ir.index] = &DownloadResult{
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
			result, err := pd.Download(ctx, ir.request)
			if err != nil {
				log.Errorln("profile download failed:", ir.request.ScreenName, "-", err)

				if errors.Is(err, syscall.ENOSPC) {
					cancel(err)
					// 把 channel 中剩余的任务标记为失败
					for remainingIr := range reqChan {
						mu.Lock()
						results[remainingIr.index] = &DownloadResult{
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
			// 把 channel 中剩余的任务标记为失败
			for ir := range reqChan {
				mu.Lock()
				results[ir.index] = &DownloadResult{
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

func (pd *ProfileDownloader) saveFile(userTitle, screenName string, fileType FileType, filePath string, data []byte, fetchedAt time.Time, logPrefix string) FileResult {
	result := FileResult{
		FileType: fileType,
		FilePath: filePath,
		NewSize:  int64(len(data)),
	}

	exists, fileInfo, err := pd.storage.FileExists(result.FilePath)
	if err != nil {
		result.Status = StatusFailed
		result.Error = err
		return result
	}

	if exists && pd.config.SkipUnchanged {
		if fileInfo.Size != result.NewSize {
			log.Debugln(logPrefix, "size changed for", screenName, "old:", fileInfo.Size, "new:", result.NewSize)
		} else {
			oldHash, err := pd.storage.ComputeFileHash(result.FilePath)
			if err != nil {
				log.Warnln(logPrefix, "hash compute failed:", screenName, "-", err)
			} else {
				newHash := ComputeDataHash(data)
				if oldHash == newHash {
					result.Status = StatusSkipped
					result.OldSize = fileInfo.Size
					log.Debugln(logPrefix, "unchanged (hash match), skipping:", screenName)
					return result
				}
				log.Debugln(logPrefix, "hash changed for", screenName)
			}
		}
	}

	if exists && pd.config.EnableVersioning {
		versionPath, err := pd.storage.CreateVersion(userTitle, screenName, fileType, result.FilePath)
		if err != nil {
			log.Warnln(logPrefix, "version backup failed:", screenName, "-", err)
		} else {
			result.VersionPath = versionPath
			result.OldSize = fileInfo.Size
			log.Debugln(logPrefix, "version backup created:", versionPath)
		}
	}

	if err := pd.storage.AtomicWrite(result.FilePath, data); err != nil {
		result.Status = StatusFailed
		result.Error = err
		log.Errorln(logPrefix, "save failed:", screenName, "-", err)
		return result
	}

	os.Chtimes(result.FilePath, time.Time{}, fetchedAt)

	if result.VersionPath != "" {
		result.Status = StatusVersioned
	} else {
		result.Status = StatusDownloaded
	}

	log.Debugln(logPrefix, "saved for", screenName)
	return result
}

func (pd *ProfileDownloader) downloadAvatar(ctx context.Context, userTitle, screenName, url string, fetchedAt time.Time) FileResult {
	ext := extractExtFromURL(url)
	filePath := pd.storage.GetFilePathWithExt(userTitle, screenName, FileTypeAvatar, ext)

	data, err := pd.fetcher.FetchAvatar(ctx, url)
	if err != nil {
		result := FileResult{
			FileType: FileTypeAvatar,
			FilePath: filePath,
			Status:   StatusFailed,
			Error:    err,
		}
		log.Errorln("avatar download failed:", screenName, "-", err)
		return result
	}

	return pd.saveFile(userTitle, screenName, FileTypeAvatar, filePath, data, fetchedAt, "avatar")
}

func (pd *ProfileDownloader) downloadBanner(ctx context.Context, userTitle, screenName, url string, fetchedAt time.Time) FileResult {
	data, ext, err := pd.fetcher.FetchBanner(ctx, url)
	if err != nil {
		result := FileResult{
			FileType: FileTypeBanner,
			Status:   StatusFailed,
			Error:    err,
		}
		log.Errorln("banner download failed:", screenName, "-", err)
		return result
	}

	filePath := pd.storage.GetFilePathWithExt(userTitle, screenName, FileTypeBanner, ext)
	return pd.saveFile(userTitle, screenName, FileTypeBanner, filePath, data, fetchedAt, "banner")
}

func (pd *ProfileDownloader) saveDescription(userTitle, screenName, description string, fetchedAt time.Time) FileResult {
	filePath := pd.storage.GetFilePath(userTitle, screenName, FileTypeDescription)
	data := []byte(description)
	return pd.saveFile(userTitle, screenName, FileTypeDescription, filePath, data, fetchedAt, "description")
}

func (pd *ProfileDownloader) saveProfileJSON(userTitle, screenName string, profile *ProfileInfo, fetchedAt time.Time) FileResult {
	filePath := pd.storage.GetFilePath(userTitle, screenName, FileTypeProfile)

	data, err := ProfileToJSON(profile)
	if err != nil {
		result := FileResult{
			FileType: FileTypeProfile,
			FilePath: filePath,
			Status:   StatusFailed,
			Error:    err,
		}
		log.Errorln("profile JSON serialize failed:", screenName, "-", err)
		return result
	}

	return pd.saveFile(userTitle, screenName, FileTypeProfile, filePath, data, fetchedAt, "profile JSON")
}
