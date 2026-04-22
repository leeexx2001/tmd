package api

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/downloading"
	"github.com/unkmonster/tmd/internal/entity"
	"github.com/unkmonster/tmd/internal/profile"
	"github.com/unkmonster/tmd/internal/twitter"
)

// executeUserDownload 执行用户下载任务
func (s *Server) executeUserDownload(task *Task) {
	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusRunning)

	data := task.Data.(*UserDownloadTaskData)
	ctx := task.Ctx

	// 获取用户信息
	user, _, err := twitter.GetUserByScreenName(ctx, s.client, data.ScreenName)
	if err != nil {
		log.Errorf("[Task:%s] Failed to get user %s: %v", task.ID, data.ScreenName, err)
		s.taskManager.SetTaskError(task.ID, err)
		return
	}

	// 更新进度
	s.taskManager.UpdateTaskProgress(task.ID, TaskProgress{
		Total:     user.MediaCount,
		Completed: 0,
		Failed:    0,
	})

	// 执行下载
	users := []*twitter.User{user}
	failed, err := downloading.BatchDownloadAny(
		ctx, s.client, s.db,
		nil, users,
		s.storePath.root, s.storePath.users,
		data.AutoFollow, s.additionalClients,
		s.dwn, s.fileWriter,
	)

	if err != nil && ctx.Err() != context.Canceled {
		log.Errorf("[Task:%s] Download failed for %s: %v", task.ID, user.ScreenName, err)
		s.taskManager.SetTaskError(task.ID, err)
		return
	}

	// 保存失败推送到 Dumper
	for _, f := range failed {
		eid, err := f.Entity.Id()
		if err != nil {
			log.Warnf("[Task:%s] Failed to get entity id: %v", task.ID, err)
			continue
		}
		s.PushFailedTweets(eid, f.Tweet)
	}

	// 下载 Profile（默认下载，除非指定 skip_profile）
	if !data.SkipProfile && ctx.Err() != context.Canceled {
		s.downloadProfile(ctx, user)
	}

	// 设置结果
	// 注意：failed 是下载失败的推文列表，不是媒体文件
	// 由于无法准确获取成功下载的媒体数量，使用推文数量作为近似值
	s.taskManager.SetTaskResult(task.ID, &TaskResult{
		Downloaded: 0, // 实际成功数无法精确统计
		Failed:     len(failed),
		Details:    []string{},
	})

	// 立即保存 Dumper（防止崩溃丢失）
	if len(failed) > 0 {
		if err := s.SaveDumper(); err != nil {
			log.Errorf("[Task:%s] Failed to save errors.json: %v", task.ID, err)
		}
	}

	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusCompleted)
	log.Infof("[Task:%s] User download completed: %s (media: %d, failed: %d)", task.ID, user.ScreenName, user.MediaCount, len(failed))
}

// executeListDownload 执行列表下载任务
func (s *Server) executeListDownload(task *Task) {
	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusRunning)

	data := task.Data.(*ListDownloadTaskData)
	ctx := task.Ctx

	// 获取列表信息
	list, err := twitter.GetLst(ctx, s.client, data.ListID)
	if err != nil {
		log.Errorf("[Task:%s] Failed to get list %d: %v", task.ID, data.ListID, err)
		s.taskManager.SetTaskError(task.ID, err)
		return
	}

	// 执行下载
	lists := []twitter.ListBase{list}
	users := []*twitter.User{}

	failed, err := downloading.BatchDownloadAny(
		ctx, s.client, s.db,
		lists, users,
		s.storePath.root, s.storePath.users,
		data.AutoFollow, s.additionalClients,
		s.dwn, s.fileWriter,
	)

	if err != nil && ctx.Err() != context.Canceled {
		log.Errorf("[Task:%s] List download failed for %d: %v", task.ID, data.ListID, err)
		s.taskManager.SetTaskError(task.ID, err)
		return
	}

	// 保存失败推送到 Dumper
	for _, f := range failed {
		eid, err := f.Entity.Id()
		if err != nil {
			log.Warnf("[Task:%s] Failed to get entity id: %v", task.ID, err)
			continue
		}
		s.PushFailedTweets(eid, f.Tweet)
	}

	// 立即保存
	if len(failed) > 0 {
		if err := s.SaveDumper(); err != nil {
			log.Errorf("[Task:%s] Failed to save errors.json: %v", task.ID, err)
		}
	}

	// 设置结果
	s.taskManager.SetTaskResult(task.ID, &TaskResult{
		Downloaded: 0, // 列表下载的计数方式不同
		Failed:     len(failed),
		Details:    []string{},
	})

	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusCompleted)
	log.Infof("[Task:%s] List download completed: %d (failed: %d)", task.ID, list.MemberCount, len(failed))
}

// executeProfileDownload 执行 Profile 下载任务
func (s *Server) executeProfileDownload(task *Task) {
	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusRunning)

	data := task.Data.(*ProfileDownloadTaskData)
	ctx := task.Ctx

	// 获取用户信息
	user, _, err := twitter.GetUserByScreenName(ctx, s.client, data.ScreenName)
	if err != nil {
		log.Errorf("[Task:%s] Failed to get user %s for profile: %v", task.ID, data.ScreenName, err)
		s.taskManager.SetTaskError(task.ID, err)
		return
	}

	// 下载 Profile
	result := s.downloadProfile(ctx, user)

	// 设置结果
	if result != nil && result.Success {
		s.taskManager.SetTaskResult(task.ID, &TaskResult{
			Downloaded: 1,
			Failed:     0,
			Details:    []string{},
		})
		s.taskManager.UpdateTaskStatus(task.ID, TaskStatusCompleted)
	} else {
		if result != nil && result.Error != nil {
			log.Errorf("[Task:%s] Profile download failed for %s: %v", task.ID, data.ScreenName, result.Error)
			s.taskManager.SetTaskError(task.ID, result.Error)
		} else {
			log.Errorf("[Task:%s] Profile download failed for %s: unknown error", task.ID, data.ScreenName)
			s.taskManager.SetTaskError(task.ID, fmt.Errorf("profile download failed"))
		}
	}
}

// executeJsonDownload 执行 JSON 下载任务
func (s *Server) executeJsonDownload(task *Task) {
	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusRunning)

	data := task.Data.(*JsonDownloadTaskData)
	ctx := task.Ctx

	results := downloading.DownloadJsonDir(
		ctx, s.client, s.storePath.root,
		s.dwn, s.fileWriter, data.Paths...,
	)

	// 检查是否被取消
	if ctx.Err() == context.Canceled {
		now := time.Now()
		task.EndedAt = &now
		s.taskManager.UpdateTaskStatus(task.ID, TaskStatusCancelled)
		return
	}

	var totalCount, successCount int
	for _, r := range results {
		totalCount += r.TweetCount
		if r.Success {
			successCount++
		}
	}

	s.taskManager.SetTaskResult(task.ID, &TaskResult{
		Downloaded: successCount,
		Failed:     len(results) - successCount,
	})

	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusCompleted)
}

// executeMarkDownloaded 执行标记已下载任务
func (s *Server) executeMarkDownloaded(task *Task) {
	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusRunning)

	data := task.Data.(*MarkDownloadedTaskData)
	ctx := task.Ctx

	// 获取用户信息
	user, _, err := twitter.GetUserByScreenName(ctx, s.client, data.ScreenName)
	if err != nil {
		log.Errorf("[Task:%s] Failed to get user %s for mark: %v", task.ID, data.ScreenName, err)
		s.taskManager.SetTaskError(task.ID, err)
		return
	}

	// 同步用户和实体
	ent, err := entity.NewUserEntity(s.db, user.Id, s.storePath.users)
	if err != nil {
		log.Errorf("[Task:%s] Failed to create entity for %s: %v", task.ID, data.ScreenName, err)
		s.taskManager.SetTaskError(task.ID, err)
		return
	}

	// 设置最新发布时间
	if data.Timestamp != nil {
		err = ent.SetLatestReleaseTime(*data.Timestamp)
	} else {
		err = ent.ClearLatestReleaseTime()
	}

	if err != nil {
		log.Errorf("[Task:%s] Failed to mark %s: %v", task.ID, data.ScreenName, err)
		s.taskManager.SetTaskError(task.ID, err)
		return
	}

	// 设置结果
	s.taskManager.SetTaskResult(task.ID, &TaskResult{
		Downloaded: 1,
		Failed:     0,
		Details:    []string{"marked as downloaded"},
	})

	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusCompleted)
}

// executeFollowingDownload 执行关注列表下载任务
func (s *Server) executeFollowingDownload(task *Task) {
	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusRunning)

	data := task.Data.(*FollowingDownloadTaskData)
	ctx := task.Ctx

	// 获取用户信息
	user, _, err := twitter.GetUserByScreenName(ctx, s.client, data.ScreenName)
	if err != nil {
		log.Errorf("[Task:%s] Failed to get user %s for following: %v", task.ID, data.ScreenName, err)
		s.taskManager.SetTaskError(task.ID, err)
		return
	}

	// 获取关注列表
	following := user.Following()
	lists := []twitter.ListBase{following}
	users := []*twitter.User{}

	// 执行下载
	failed, err := downloading.BatchDownloadAny(
		ctx, s.client, s.db,
		lists, users,
		s.storePath.root, s.storePath.users,
		data.AutoFollow, s.additionalClients,
		s.dwn, s.fileWriter,
	)

	if err != nil && ctx.Err() != context.Canceled {
		log.Errorf("[Task:%s] Following download failed for %s: %v", task.ID, data.ScreenName, err)
		s.taskManager.SetTaskError(task.ID, err)
		return
	}

	// 保存失败推送到 Dumper
	for _, f := range failed {
		eid, err := f.Entity.Id()
		if err != nil {
			log.Warnf("[Task:%s] Failed to get entity id: %v", task.ID, err)
			continue
		}
		s.PushFailedTweets(eid, f.Tweet)
	}

	// 下载 Profile（默认下载，除非指定 skip_profile）
	if !data.SkipProfile && ctx.Err() != context.Canceled {
		s.batchDownloadListProfiles(ctx, following)
	}

	// 立即保存
	if len(failed) > 0 {
		if err := s.SaveDumper(); err != nil {
			log.Errorf("[Task:%s] Failed to save errors.json: %v", task.ID, err)
		}
	}

	// 设置结果
	s.taskManager.SetTaskResult(task.ID, &TaskResult{
		Downloaded: 0, // 关注列表下载的计数方式不同
		Failed:     len(failed),
		Details:    []string{},
	})

	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusCompleted)
}

// executeListProfileDownload 执行列表 Profile 下载任务
func (s *Server) executeListProfileDownload(task *Task) {
	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusRunning)

	data := task.Data.(*ListProfileDownloadTaskData)
	ctx := task.Ctx

	// 获取列表信息
	list, err := twitter.GetLst(ctx, s.client, data.ListID)
	if err != nil {
		log.Errorf("[Task:%s] Failed to get list %d for profile: %v", task.ID, data.ListID, err)
		s.taskManager.SetTaskError(task.ID, err)
		return
	}

	// 下载列表成员 Profile
	s.batchDownloadListProfiles(ctx, list)

	// 设置结果
	s.taskManager.SetTaskResult(task.ID, &TaskResult{
		Downloaded: 1,
		Failed:     0,
		Details:    []string{"list profile download initiated"},
	})

	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusCompleted)
}

// batchDownloadListProfiles 批量下载列表成员 Profile
func (s *Server) batchDownloadListProfiles(ctx context.Context, list twitter.ListBase) {
	storage, err := profile.NewFileStorageManager(s.storePath.users)
	if err != nil {
		log.Warnln("Failed to create profile storage:", err)
		return
	}
	storage.SetVersionManager(s.versionManager)

	clients := append([]*resty.Client{s.client}, s.additionalClients...)
	pd := profile.NewProfileDownloaderWithDB(nil, storage, clients, s.db, s.dwn, s.fileWriter)

	// 获取列表成员
	result, err := list.GetMembers(ctx, s.client)
	if err != nil {
		log.WithError(err).Warnln("Failed to get list members")
		return
	}

	// 创建下载请求
	requests := make([]profile.DownloadRequest, 0, len(result.Users))
	for _, member := range result.Users {
		requests = append(requests, profile.DownloadRequest{
			ScreenName:  member.ScreenName,
			UserTitle:   member.Title(),
			Name:        member.Name,
			UserID:      member.Id,
			AvatarURL:   member.AvatarURL,
			BannerURL:   member.BannerURL,
			Description: member.Description,
			Location:    member.Location,
			URL:         member.URL,
			Verified:    member.Verified,
			Protected:   member.IsProtected,
			CreatedAt:   member.CreatedAt,
		})
	}

	// 去重
	seen := make(map[string]bool)
	uniqueRequests := make([]profile.DownloadRequest, 0)
	for _, req := range requests {
		if !seen[req.ScreenName] {
			seen[req.ScreenName] = true
			uniqueRequests = append(uniqueRequests, req)
		}
	}

	// 批量下载
	pd.DownloadMultiple(ctx, uniqueRequests)
}

// executeBatchDownload 执行批量下载任务
func (s *Server) executeBatchDownload(task *Task) {
	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusRunning)

	data := task.Data.(*BatchDownloadTaskData)
	ctx := task.Ctx

	var totalFailed int
	var failedQueries int // 记录查询失败的

	// ===== 阶段1: 收集所有用户 =====
	allUsers := make([]*twitter.User, 0, len(data.Users))
	for _, screenName := range data.Users {
		if ctx.Err() == context.Canceled {
			break
		}

		user, _, err := twitter.GetUserByScreenName(ctx, s.client, screenName)
		if err != nil {
			log.Warnf("[Task:%s] Failed to get user %s: %v", task.ID, screenName, err)
			failedQueries++
			continue
		}
		allUsers = append(allUsers, user)
	}

	// ===== 阶段2: 收集所有列表 =====
	allLists := make([]twitter.ListBase, 0, len(data.Lists))
	for _, listID := range data.Lists {
		if ctx.Err() == context.Canceled {
			break
		}

		list, err := twitter.GetLst(ctx, s.client, listID)
		if err != nil {
			log.Warnf("[Task:%s] Failed to get list %d: %v", task.ID, listID, err)
			failedQueries++
			continue
		}
		allLists = append(allLists, list)
	}

	// ===== 阶段3: 一次性批量下载 =====
	if len(allUsers) > 0 || len(allLists) > 0 {
		failed, err := downloading.BatchDownloadAny(
			ctx, s.client, s.db,
			allLists, allUsers,
			s.storePath.root, s.storePath.users,
			data.AutoFollow, s.additionalClients,
			s.dwn, s.fileWriter,
		)

		if err != nil {
			log.Errorf("[Task:%s] Batch download failed: %v", task.ID, err)
			totalFailed += len(allUsers) + len(allLists) // 整体失败，全部算失败
		} else {
			totalFailed += len(failed)
			// 保存失败推文到 Dumper
			for _, f := range failed {
				eid, err := f.Entity.Id()
				if err != nil {
					log.Warnf("[Task:%s] Failed to get entity id: %v", task.ID, err)
					continue
				}
				s.PushFailedTweets(eid, f.Tweet)
			}
		}

		// 立即保存 Dumper
		if s.GetFailedTweetsCount() > 0 {
			if err := s.SaveDumper(); err != nil {
				log.Errorf("[Task:%s] Failed to save errors.json: %v", task.ID, err)
			}
		}
	}

	// ===== 阶段4: 下载 Profile（保持循环，因为是独立操作）=====
	if !data.SkipProfile {
		for _, user := range allUsers {
			if ctx.Err() == context.Canceled {
				break
			}
			s.downloadProfile(ctx, user)
		}

		for _, list := range allLists {
			if ctx.Err() == context.Canceled {
				break
			}
			s.batchDownloadListProfiles(ctx, list)
		}
	}

	totalFailed += failedQueries

	// 设置结果
	s.taskManager.SetTaskResult(task.ID, &TaskResult{
		Downloaded: len(allUsers) + len(allLists) - totalFailed,
		Failed:     totalFailed,
		Details:    []string{},
	})

	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusCompleted)
	log.Infof("[Task:%s] Batch download completed: users=%d/%d, lists=%d/%d, failed=%d",
		task.ID, len(allUsers), len(data.Users), len(allLists), len(data.Lists), totalFailed)
}

// downloadProfile 下载用户 Profile
func (s *Server) downloadProfile(ctx context.Context, user *twitter.User) *profile.DownloadResult {
	storage, err := profile.NewFileStorageManager(s.storePath.users)
	if err != nil {
		log.Warnln("Failed to create profile storage:", err)
		return &profile.DownloadResult{
			ScreenName: user.ScreenName,
			Success:    false,
			Error:      err,
		}
	}
	storage.SetVersionManager(s.versionManager)

	clients := append([]*resty.Client{s.client}, s.additionalClients...)
	pd := profile.NewProfileDownloaderWithDB(nil, storage, clients, s.db, s.dwn, s.fileWriter)

	req := profile.DownloadRequest{
		ScreenName:  user.ScreenName,
		UserTitle:   user.Title(),
		Name:        user.Name,
		UserID:      user.Id,
		AvatarURL:   user.AvatarURL,
		BannerURL:   user.BannerURL,
		Description: user.Description,
		Location:    user.Location,
		URL:         user.URL,
		Verified:    user.Verified,
		Protected:   user.IsProtected,
		CreatedAt:   user.CreatedAt,
	}

	result, _ := pd.Download(ctx, req)
	if result != nil && result.Success {
		var avatarFile, bannerFile string
		for _, f := range result.Files {
			if f.Status == profile.StatusDownloaded {
				switch f.FileType {
				case profile.FileTypeAvatar:
					avatarFile = f.FilePath
				case profile.FileTypeBanner:
					bannerFile = f.FilePath
				}
			}
		}
		log.Infof("[Profile] Downloaded: %s (avatar: %s, banner: %s)", user.ScreenName, avatarFile, bannerFile)
	} else if result != nil && result.Error != nil {
		log.Warnf("[Profile] Failed to download: %s - %v", user.ScreenName, result.Error)
	}
	return result
}

// executeRetry 执行重试任务
func (s *Server) executeRetry(task *Task) {
	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusRunning)

	data := task.Data.(*RetryTaskData)
	ctx := task.Ctx

	beforeCount := s.GetFailedTweetsCount()

	if data.NoRetry {
		// 仅清理，不重试
		s.dumperMu.Lock()
		s.dumper.Clear()
		s.dumperMu.Unlock()
		if err := s.SaveDumper(); err != nil {
			log.Errorf("[Task:%s] Failed to clear errors.json: %v", task.ID, err)
		}

		s.taskManager.SetTaskResult(task.ID, &TaskResult{
			Downloaded: 0,
			Failed:     0,
			Details:    []string{"errors.json cleared"},
		})
		s.taskManager.UpdateTaskStatus(task.ID, TaskStatusCompleted)
		return
	}

	// 执行重试
	if err := s.RetryFailedTweets(ctx); err != nil {
		log.Errorf("[Task:%s] Retry failed: %v", task.ID, err)
		s.taskManager.SetTaskError(task.ID, err)
		return
	}

	afterCount := s.GetFailedTweetsCount()

	s.taskManager.SetTaskResult(task.ID, &TaskResult{
		Downloaded: beforeCount - afterCount,
		Failed:     afterCount,
		Details:    []string{fmt.Sprintf("Retried %d tweets", beforeCount)},
	})

	s.taskManager.UpdateTaskStatus(task.ID, TaskStatusCompleted)
	log.Infof("[Task:%s] Retry completed: %d -> %d failed tweets", task.ID, beforeCount, afterCount)
}
