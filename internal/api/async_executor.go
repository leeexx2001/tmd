package api

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/unkmonster/tmd/internal/service"
	"github.com/unkmonster/tmd/internal/twitter"
)

// AsyncExecutor 异步执行器
type AsyncExecutor struct {
	taskManager *TaskManager
	server      *Server
}

// NewAsyncExecutor 创建异步执行器
func NewAsyncExecutor(tm *TaskManager, server *Server) *AsyncExecutor {
	return &AsyncExecutor{
		taskManager: tm,
		server:      server,
	}
}

// ExecuteTask 异步执行任务
func (ae *AsyncExecutor) ExecuteTask(taskID string, taskType TaskType, data interface{}) {
	task, ok := ae.taskManager.GetTask(taskID)
	if !ok {
		log.Errorf("[AsyncExecutor] Task not found: %s", taskID)
		return
	}

	// 更新状态为运行中
	ae.taskManager.UpdateTaskStatus(taskID, TaskStatusRunning)

	// 在 goroutine 中执行
	go func() {
		log.Infof("[Task:%s] Starting async execution", taskID)

		var err error
		switch taskType {
		case TaskTypeUserDownload:
			err = ae.executeUserDownload(task.Ctx, data)
		case TaskTypeListDownload:
			err = ae.executeListDownload(task.Ctx, data)
		case TaskTypeFollowingDownload:
			err = ae.executeFollowingDownload(task.Ctx, data)
		case TaskTypeProfileDownload:
			err = ae.executeProfileDownload(task.Ctx, data)
		case TaskTypeMarkDownloaded:
			err = ae.executeMarkDownloaded(task.Ctx, data)
		case TaskTypeJsonDownload:
			err = ae.executeJsonDownload(task.Ctx, data)
		case TaskTypeBatchDownload:
			err = ae.executeBatchDownload(task.Ctx, data)
		case TaskTypeListProfile:
			err = ae.executeListProfile(task.Ctx, data)
		default:
			err = fmt.Errorf("unknown task type: %s", taskType)
		}

		if err != nil {
			log.Errorf("[Task:%s] Execution failed: %v", taskID, err)
			ae.taskManager.SetTaskError(taskID, err)
		} else {
			log.Infof("[Task:%s] Execution completed", taskID)
			ae.taskManager.UpdateTaskStatus(taskID, TaskStatusCompleted)
		}
	}()
}

func (ae *AsyncExecutor) executeUserDownload(ctx context.Context, data interface{}) error {
	d, ok := data.(*UserDownloadTaskData)
	if !ok {
		return fmt.Errorf("invalid data type")
	}

	// 获取用户信息
	user, _, err := twitter.GetUserByScreenName(ctx, ae.server.client, d.ScreenName)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	req := &service.DownloadUsersRequest{
		Users:       []*twitter.User{user},
		AutoFollow:  d.AutoFollow,
		NoRetry:     d.NoRetry,
		SkipProfile: d.SkipProfile,
	}
	return ae.server.services.Download.ExecuteDownloadUsers(ctx, req)
}

func (ae *AsyncExecutor) executeListDownload(ctx context.Context, data interface{}) error {
	d, ok := data.(*ListDownloadTaskData)
	if !ok {
		return fmt.Errorf("invalid data type")
	}

	list, err := twitter.GetLst(ctx, ae.server.client, d.ListID)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}

	req := &service.DownloadListsRequest{
		Lists:       []twitter.ListBase{list},
		AutoFollow:  d.AutoFollow,
		NoRetry:     d.NoRetry,
		SkipProfile: d.SkipProfile,
	}
	return ae.server.services.Download.ExecuteDownloadLists(ctx, req)
}

func (ae *AsyncExecutor) executeFollowingDownload(ctx context.Context, data interface{}) error {
	d, ok := data.(*FollowingDownloadTaskData)
	if !ok {
		return fmt.Errorf("invalid data type")
	}

	user, _, err := twitter.GetUserByScreenName(ctx, ae.server.client, d.ScreenName)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	req := &service.DownloadFollowingRequest{
		Users:       []*twitter.User{user},
		AutoFollow:  d.AutoFollow,
		NoRetry:     d.NoRetry,
		SkipProfile: d.SkipProfile,
	}
	return ae.server.services.Download.ExecuteDownloadFollowing(ctx, req)
}

func (ae *AsyncExecutor) executeProfileDownload(ctx context.Context, data interface{}) error {
	d, ok := data.(*ProfileDownloadTaskData)
	if !ok {
		return fmt.Errorf("invalid data type")
	}

	user, _, err := twitter.GetUserByScreenName(ctx, ae.server.client, d.ScreenName)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	req := &service.DownloadProfilesRequest{
		Users: []*twitter.User{user},
	}
	return ae.server.services.Download.ExecuteDownloadProfiles(ctx, req)
}

func (ae *AsyncExecutor) executeMarkDownloaded(ctx context.Context, data interface{}) error {
	d, ok := data.(*MarkDownloadedTaskData)
	if !ok {
		return fmt.Errorf("invalid data type")
	}

	user, _, err := twitter.GetUserByScreenName(ctx, ae.server.client, d.ScreenName)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	req := &service.MarkDownloadedRequest{
		Users:     []*twitter.User{user},
		Timestamp: d.Timestamp,
	}
	return ae.server.services.Mark.ExecuteMarkDownloaded(ctx, req)
}

func (ae *AsyncExecutor) executeJsonDownload(ctx context.Context, data interface{}) error {
	d, ok := data.(*JsonDownloadTaskData)
	if !ok {
		return fmt.Errorf("invalid data type")
	}

	req := &service.DownloadJsonRequest{
		Paths:   d.Paths,
		NoRetry: d.NoRetry,
	}
	return ae.server.services.Json.ExecuteDownloadJson(ctx, req)
}

func (ae *AsyncExecutor) executeBatchDownload(ctx context.Context, data interface{}) error {
	d, ok := data.(*BatchDownloadTaskData)
	if !ok {
		return fmt.Errorf("invalid data type")
	}

	// 获取所有用户信息
	users := make([]*twitter.User, 0, len(d.Users))
	for _, screenName := range d.Users {
		user, _, err := twitter.GetUserByScreenName(ctx, ae.server.client, screenName)
		if err != nil {
			log.WithError(err).Warnf("Failed to get user %s", screenName)
			continue
		}
		users = append(users, user)
	}

	// 获取所有列表信息
	lists := make([]twitter.ListBase, 0, len(d.Lists))
	for _, listID := range d.Lists {
		list, err := twitter.GetLst(ctx, ae.server.client, listID)
		if err != nil {
			log.WithError(err).Warnf("Failed to get list %d", listID)
			continue
		}
		lists = append(lists, list)
	}

	// 先下载用户
	if len(users) > 0 {
		req := &service.DownloadUsersRequest{
			Users:       users,
			AutoFollow:  d.AutoFollow,
			NoRetry:     d.NoRetry,
			SkipProfile: d.SkipProfile,
		}
		if err := ae.server.services.Download.ExecuteDownloadUsers(ctx, req); err != nil {
			return err
		}
	}

	// 再下载列表
	if len(lists) > 0 {
		req := &service.DownloadListsRequest{
			Lists:       lists,
			AutoFollow:  d.AutoFollow,
			NoRetry:     d.NoRetry,
			SkipProfile: d.SkipProfile,
		}
		if err := ae.server.services.Download.ExecuteDownloadLists(ctx, req); err != nil {
			return err
		}
	}

	return nil
}

func (ae *AsyncExecutor) executeListProfile(ctx context.Context, data interface{}) error {
	d, ok := data.(*BatchDownloadTaskData)
	if !ok {
		return fmt.Errorf("invalid data type")
	}

	// 获取所有用户信息
	users := make([]*twitter.User, 0, len(d.Users))
	for _, screenName := range d.Users {
		user, _, err := twitter.GetUserByScreenName(ctx, ae.server.client, screenName)
		if err != nil {
			log.WithError(err).Warnf("Failed to get user %s", screenName)
			continue
		}
		users = append(users, user)
	}

	req := &service.DownloadProfilesRequest{
		Users: users,
	}
	return ae.server.services.Download.ExecuteDownloadProfiles(ctx, req)
}
