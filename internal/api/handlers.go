package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/unkmonster/tmd/internal/twitter"
)

// handleUserDownload 处理用户推文下载
func (s *Server) handleUserDownload(w http.ResponseWriter, r *http.Request, screenName string) {
	var req UserDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// 空请求体也允许，使用默认值
		req = UserDownloadRequest{}
	}

	// 创建任务
	task := s.taskManager.CreateTask(TaskTypeUserDownload, &UserDownloadTaskData{
		ScreenName:  screenName,
		AutoFollow:  req.AutoFollow,
		SkipProfile: req.SkipProfile,
		NoRetry:     req.NoRetry,
	})

	// 异步执行任务
	go s.executeUserDownload(task)

	// 获取用户信息（同步快速检查）
	ctx := r.Context()
	user, _, err := twitter.GetUserByScreenName(ctx, s.client, screenName)
	if err != nil {
		// 用户不存在也可以继续，异步任务会处理
		writeJSON(w, http.StatusAccepted, Response{
			Success: true,
			Data: map[string]interface{}{
				"task_id": task.ID,
				"status":  string(task.Status),
				"message": "Download task queued",
			},
		})
		return
	}

	writeJSON(w, http.StatusAccepted, Response{
		Success: true,
		Data: map[string]interface{}{
			"task_id": task.ID,
			"status":  string(task.Status),
			"user":    UserInfo{ID: user.Id, ScreenName: user.ScreenName, Name: user.Name},
			"message": "Download task queued successfully",
		},
	})
}

// handleUserProfile 处理用户 Profile 下载
func (s *Server) handleUserProfile(w http.ResponseWriter, _ *http.Request, screenName string) {
	// 创建任务
	task := s.taskManager.CreateTask(TaskTypeProfileDownload, &ProfileDownloadTaskData{
		ScreenName: screenName,
	})

	// 异步执行任务
	go s.executeProfileDownload(task)

	writeJSON(w, http.StatusAccepted, Response{
		Success: true,
		Data: map[string]interface{}{
			"task_id": task.ID,
			"status":  string(task.Status),
			"message": "Profile download task queued",
		},
	})
}

// handleUserMark 处理用户标记为已下载
func (s *Server) handleUserMark(w http.ResponseWriter, r *http.Request, screenName string) {
	var req MarkUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = MarkUserRequest{}
	}

	// 创建任务
	task := s.taskManager.CreateTask(TaskTypeMarkDownloaded, &MarkDownloadedTaskData{
		ScreenName: screenName,
		Timestamp:  req.Timestamp,
	})

	// 异步执行任务
	go s.executeMarkDownloaded(task)

	writeJSON(w, http.StatusAccepted, Response{
		Success: true,
		Data: map[string]interface{}{
			"task_id": task.ID,
			"status":  string(task.Status),
			"message": "Mark downloaded task queued",
		},
	})
}

// handleListDownload 处理列表下载
func (s *Server) handleListDownload(w http.ResponseWriter, r *http.Request, listIDStr string) {
	listID, err := strconv.ParseUint(listIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid list ID")
		return
	}

	var req ListDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = ListDownloadRequest{}
	}

	// 创建任务
	task := s.taskManager.CreateTask(TaskTypeListDownload, &ListDownloadTaskData{
		ListID:      listID,
		AutoFollow:  req.AutoFollow,
		SkipProfile: req.SkipProfile,
		NoRetry:     req.NoRetry,
	})

	// 异步执行任务
	go s.executeListDownload(task)

	writeJSON(w, http.StatusAccepted, Response{
		Success: true,
		Data: map[string]interface{}{
			"task_id": task.ID,
			"status":  string(task.Status),
			"list_id": listID,
			"message": "List download task queued",
		},
	})
}

// UserDownloadTaskData 用户下载任务数据
type UserDownloadTaskData struct {
	ScreenName  string
	AutoFollow  bool
	SkipProfile bool
	NoRetry     bool
}

// ListDownloadTaskData 列表下载任务数据
type ListDownloadTaskData struct {
	ListID      uint64
	AutoFollow  bool
	SkipProfile bool
	NoRetry     bool
}

// JsonDownloadTaskData JSON 下载任务数据
type JsonDownloadTaskData struct {
	Paths   []string
	NoRetry bool
}

// ProfileDownloadTaskData Profile 下载任务数据
type ProfileDownloadTaskData struct {
	ScreenName string
}

// MarkDownloadedTaskData 标记已下载任务数据
type MarkDownloadedTaskData struct {
	ScreenName string
	Timestamp  *time.Time
}

// FollowingDownloadTaskData 关注列表下载任务数据
type FollowingDownloadTaskData struct {
	ScreenName  string
	AutoFollow  bool
	SkipProfile bool
	NoRetry     bool
}

// ListProfileDownloadTaskData 列表 Profile 下载任务数据
type ListProfileDownloadTaskData struct {
	ListID uint64
}

// BatchDownloadTaskData 批量下载任务数据
type BatchDownloadTaskData struct {
	Users       []string
	Lists       []uint64
	AutoFollow  bool
	SkipProfile bool
	NoRetry     bool
}

// handleUserFollowingDownload 处理用户关注列表下载
func (s *Server) handleUserFollowingDownload(w http.ResponseWriter, r *http.Request, screenName string) {
	var req UserDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = UserDownloadRequest{}
	}

	// 创建任务
	task := s.taskManager.CreateTask(TaskTypeFollowingDownload, &FollowingDownloadTaskData{
		ScreenName:  screenName,
		AutoFollow:  req.AutoFollow,
		SkipProfile: req.SkipProfile,
		NoRetry:     req.NoRetry,
	})

	// 异步执行任务
	go s.executeFollowingDownload(task)

	// 获取用户信息（同步快速检查）
	ctx := r.Context()
	user, _, err := twitter.GetUserByScreenName(ctx, s.client, screenName)
	if err != nil {
		writeJSON(w, http.StatusAccepted, Response{
			Success: true,
			Data: map[string]interface{}{
				"task_id": task.ID,
				"status":  string(task.Status),
				"message": "Following download task queued",
			},
		})
		return
	}

	writeJSON(w, http.StatusAccepted, Response{
		Success: true,
		Data: map[string]interface{}{
			"task_id": task.ID,
			"status":  string(task.Status),
			"user":    UserInfo{ID: user.Id, ScreenName: user.ScreenName, Name: user.Name},
			"message": "Following download task queued successfully",
		},
	})
}

// handleListProfile 处理列表 Profile 下载
func (s *Server) handleListProfile(w http.ResponseWriter, _ *http.Request, listIDStr string) {
	listID, err := strconv.ParseUint(listIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid list ID")
		return
	}

	// 创建任务
	task := s.taskManager.CreateTask(TaskTypeListProfileDownload, &ListProfileDownloadTaskData{
		ListID: listID,
	})

	// 异步执行任务
	go s.executeListProfileDownload(task)

	writeJSON(w, http.StatusAccepted, Response{
		Success: true,
		Data: map[string]interface{}{
			"task_id": task.ID,
			"status":  string(task.Status),
			"list_id": listID,
			"message": "List profile download task queued",
		},
	})
}
