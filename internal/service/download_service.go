package service

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/config"
	"github.com/unkmonster/tmd/internal/downloader"
	"github.com/unkmonster/tmd/internal/downloading"
	"github.com/unkmonster/tmd/internal/path"
	"github.com/unkmonster/tmd/internal/twitter"
)

// DownloadService 下载服务
type DownloadService struct {
	client            *resty.Client
	additionalClients []*resty.Client
	db                *sqlx.DB
	conf              *config.Config
	appRootPath       string
	profileService    *ProfileService
}

// NewDownloadService 创建下载服务
func NewDownloadService(client *resty.Client, additionalClients []*resty.Client, db *sqlx.DB, conf *config.Config, appRootPath string, profileService *ProfileService) *DownloadService {
	return &DownloadService{
		client:            client,
		additionalClients: additionalClients,
		db:                db,
		conf:              conf,
		appRootPath:       appRootPath,
		profileService:    profileService,
	}
}

// DownloadUsersRequest 下载用户请求
type DownloadUsersRequest struct {
	Users       []*twitter.User
	AutoFollow  bool
	NoRetry     bool
	SkipProfile bool
}

// DownloadListsRequest 下载列表请求
type DownloadListsRequest struct {
	Lists       []twitter.ListBase
	AutoFollow  bool
	NoRetry     bool
	SkipProfile bool
}

// DownloadFollowingRequest 下载关注列表请求
type DownloadFollowingRequest struct {
	Users       []*twitter.User
	AutoFollow  bool
	NoRetry     bool
	SkipProfile bool
}

// ExecuteDownloadUsers 执行用户下载
func (s *DownloadService) ExecuteDownloadUsers(ctx context.Context, req *DownloadUsersRequest) error {
	pathHelper, err := path.NewStorePath(s.conf.RootPath)
	if err != nil {
		return fmt.Errorf("failed to create store path: %w", err)
	}

	// 创建下载器
	versionManager := downloader.NewVersionManagerWithWriter(pathHelper.Data, nil)
	fileWriter := downloader.NewFileWriter(versionManager)
	versionManager.SetFileWriter(fileWriter)
	dwn := downloader.NewDownloader(fileWriter)

	// 创建 Dumper
	dumper := downloading.NewDumper()
	_ = dumper.Load(pathHelper.ErrorJ)

	// 保存 Dumper 的 defer（需要在函数返回前执行）
	defer func() {
		if dumper.Count() > 0 {
			dumper.Dump(pathHelper.ErrorJ)
			log.Infof("%d tweets have been dumped", dumper.Count())
		}
	}()

	// 执行批量下载
	failed, err := downloading.BatchDownloadAny(
		ctx, s.client, s.db,
		nil, req.Users,
		pathHelper.Root, pathHelper.Users,
		req.AutoFollow, s.additionalClients,
		dwn, fileWriter,
	)

	if err != nil {
		return err
	}

	// 保存失败推文
	for _, f := range failed {
		eid, err := f.Entity.Id()
		if err != nil {
			log.Warnln("failed to get entity id:", err)
			continue
		}
		dumper.Push(eid, f.Tweet)
	}

	// 下载 Profile
	if !req.SkipProfile {
		s.ExecuteDownloadProfiles(ctx, &DownloadProfilesRequest{Users: req.Users})
	}

	// 重试失败的
	if !req.NoRetry {
		downloading.RetryFailedTweets(ctx, dumper, s.db, s.client, dwn, fileWriter)
	}

	return nil
}

// ExecuteDownloadLists 执行列表下载
func (s *DownloadService) ExecuteDownloadLists(ctx context.Context, req *DownloadListsRequest) error {
	pathHelper, err := path.NewStorePath(s.conf.RootPath)
	if err != nil {
		return fmt.Errorf("failed to create store path: %w", err)
	}

	// 创建下载器
	versionManager := downloader.NewVersionManagerWithWriter(pathHelper.Data, nil)
	fileWriter := downloader.NewFileWriter(versionManager)
	versionManager.SetFileWriter(fileWriter)
	dwn := downloader.NewDownloader(fileWriter)

	// 创建 Dumper
	dumper := downloading.NewDumper()
	_ = dumper.Load(pathHelper.ErrorJ)

	// 保存 Dumper 的 defer
	defer func() {
		if dumper.Count() > 0 {
			dumper.Dump(pathHelper.ErrorJ)
			log.Infof("%d tweets have been dumped", dumper.Count())
		}
	}()

	// 执行批量下载
	failed, err := downloading.BatchDownloadAny(
		ctx, s.client, s.db,
		req.Lists, nil,
		pathHelper.Root, pathHelper.Users,
		req.AutoFollow, s.additionalClients,
		dwn, fileWriter,
	)

	if err != nil {
		return err
	}

	// 保存失败推文
	for _, f := range failed {
		eid, err := f.Entity.Id()
		if err != nil {
			log.Warnln("failed to get entity id:", err)
			continue
		}
		dumper.Push(eid, f.Tweet)
	}

	// 重试失败的
	if !req.NoRetry {
		downloading.RetryFailedTweets(ctx, dumper, s.db, s.client, dwn, fileWriter)
	}

	return nil
}

// ExecuteDownloadFollowing 执行关注列表下载
func (s *DownloadService) ExecuteDownloadFollowing(ctx context.Context, req *DownloadFollowingRequest) error {
	// 将用户转换为关注列表
	lists := make([]twitter.ListBase, 0, len(req.Users))
	for _, user := range req.Users {
		lists = append(lists, user.Following())
	}

	return s.ExecuteDownloadLists(ctx, &DownloadListsRequest{
		Lists:       lists,
		AutoFollow:  req.AutoFollow,
		NoRetry:     req.NoRetry,
		SkipProfile: req.SkipProfile,
	})
}

// DownloadProfilesRequest 下载 Profile 请求
type DownloadProfilesRequest struct {
	Users []*twitter.User
}

// ExecuteDownloadProfiles 执行 Profile 下载
func (s *DownloadService) ExecuteDownloadProfiles(ctx context.Context, req *DownloadProfilesRequest) error {
	return s.profileService.ExecuteDownloadProfiles(ctx, &DownloadProfilesRequest{
		Users: req.Users,
	})
}

// DownloadListProfilesRequest 下载列表 Profile 请求
type DownloadListProfilesRequest struct {
	ListID uint64
}

// ExecuteDownloadListProfiles 执行列表 Profile 下载
func (s *DownloadService) ExecuteDownloadListProfiles(ctx context.Context, req *DownloadListProfilesRequest) error {
	// 获取列表信息
	list, err := twitter.GetLst(ctx, s.client, req.ListID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	// 获取列表成员
	membersResult, err := list.GetMembers(ctx, s.client)
	if err != nil {
		return fmt.Errorf("failed to get list members: %w", err)
	}

	users := make([]*twitter.User, len(membersResult.Users))
	for i, u := range membersResult.Users {
		users[i] = &twitter.User{Id: u.Id, ScreenName: u.ScreenName, Name: u.Name}
	}

	return s.ExecuteDownloadProfiles(ctx, &DownloadProfilesRequest{Users: users})
}
