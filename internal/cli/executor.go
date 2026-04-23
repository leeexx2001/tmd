package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"

	"github.com/unkmonster/tmd/internal/config"
	"github.com/unkmonster/tmd/internal/service"
	"github.com/unkmonster/tmd/internal/twitter"
)

// Dependencies 执行依赖
type Dependencies struct {
	Client            *resty.Client
	AdditionalClients []*resty.Client
	DB                *sqlx.DB
	Conf              *config.Config
	AppRootPath       string
}

// Execute 执行 CLI 命令
func Execute(ctx context.Context, args []string, deps *Dependencies) error {
	// 1. 解析参数
	_, cfg, err := ParseArgs(args)
	if err != nil {
		return fmt.Errorf("failed to parse args: %w", err)
	}

	// 2. 创建服务
	services := service.NewServices(deps.Client, deps.AdditionalClients, deps.DB, deps.Conf, deps.AppRootPath)

	// 3. 根据参数调用相应的服务
	if len(cfg.UsrArgs.ID) > 0 || len(cfg.UsrArgs.ScreenName) > 0 {
		users, err := cfg.UsrArgs.ResolveUsers(ctx, deps.Client, deps.DB)
		if err != nil {
			return err
		}
		req := &service.DownloadUsersRequest{
			Users:       users,
			AutoFollow:  cfg.AutoFollow,
			NoRetry:     cfg.NoRetry,
			SkipProfile: cfg.NoProfile,
		}
		if err := services.Download.ExecuteDownloadUsers(ctx, req); err != nil {
			return err
		}
	}

	if len(cfg.ListArgs.ID) > 0 {
		lists, err := cfg.ListArgs.ResolveLists(ctx, deps.Client)
		if err != nil {
			return err
		}
		req := &service.DownloadListsRequest{
			Lists:       lists,
			AutoFollow:  cfg.AutoFollow,
			NoRetry:     cfg.NoRetry,
			SkipProfile: cfg.NoProfile,
		}
		if err := services.Download.ExecuteDownloadLists(ctx, req); err != nil {
			return err
		}
	}

	if len(cfg.FollArgs.ID) > 0 || len(cfg.FollArgs.ScreenName) > 0 {
		users, err := cfg.FollArgs.ResolveUsers(ctx, deps.Client, deps.DB)
		if err != nil {
			return err
		}
		req := &service.DownloadFollowingRequest{
			Users:       users,
			AutoFollow:  cfg.AutoFollow,
			NoRetry:     cfg.NoRetry,
			SkipProfile: cfg.NoProfile,
		}
		if err := services.Download.ExecuteDownloadFollowing(ctx, req); err != nil {
			return err
		}
	}

	if len(cfg.ProfileUsers.ID) > 0 || len(cfg.ProfileUsers.ScreenName) > 0 {
		users, err := cfg.ProfileUsers.ResolveUsers(ctx, deps.Client, deps.DB)
		if err != nil {
			return err
		}
		req := &service.DownloadProfilesRequest{
			Users: users,
		}
		if err := services.Download.ExecuteDownloadProfiles(ctx, req); err != nil {
			return err
		}
	}

	if len(cfg.ProfileList.ID) > 0 {
		for _, id := range cfg.ProfileList.ID {
			req := &service.DownloadListProfilesRequest{
				ListID: id,
			}
			if err := services.Download.ExecuteDownloadListProfiles(ctx, req); err != nil {
				return err
			}
		}
	}

	if len(cfg.JsonArgs.Paths) > 0 {
		req := &service.DownloadJsonRequest{
			Paths:   cfg.JsonArgs.Paths,
			NoRetry: cfg.NoRetry,
		}
		if err := services.Json.ExecuteDownloadJson(ctx, req); err != nil {
			return err
		}
	}

	if cfg.MarkDownloaded && (len(cfg.UsrArgs.ID) > 0 || len(cfg.UsrArgs.ScreenName) > 0) {
		users, err := cfg.UsrArgs.ResolveUsers(ctx, deps.Client, deps.DB)
		if err != nil {
			return err
		}
		var ts *time.Time
		if cfg.MarkTime != "" {
			t, err := time.Parse("2006-01-02T15:04:05", cfg.MarkTime)
			if err != nil {
				return fmt.Errorf("invalid mark time format: %w", err)
			}
			ts = &t
		}
		req := &service.MarkDownloadedRequest{
			Users:     users,
			Timestamp: ts,
		}
		if err := services.Mark.ExecuteMarkDownloaded(ctx, req); err != nil {
			return err
		}
	}

	return nil
}

// SetClientLogger 设置客户端日志
func SetClientLogger(client *resty.Client, out io.Writer) {
	// 此函数保留以保持向后兼容
	// 实际实现已在其他地方
}

// Task 任务结构体（保留以保持兼容性）
type Task struct {
	Users []*twitter.User
	Lists []twitter.ListBase
}

// PrintTask 打印任务（保留以保持兼容性）
func PrintTask(task *Task) {
	// 此函数保留以保持向后兼容
}
