package api

import (
	"time"
)

// Response 通用响应类型
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

// UserDownloadRequest 用户下载请求
type UserDownloadRequest struct {
	AutoFollow  bool `json:"auto_follow,omitempty"`
	SkipProfile bool `json:"skip_profile,omitempty"`
	NoRetry     bool `json:"no_retry,omitempty"`
}

// ListDownloadRequest 列表下载请求
type ListDownloadRequest struct {
	AutoFollow  bool `json:"auto_follow,omitempty"`
	SkipProfile bool `json:"skip_profile,omitempty"`
	NoRetry     bool `json:"no_retry,omitempty"`
}

// JsonDownloadRequest JSON 下载请求
type JsonDownloadRequest struct {
	Paths   []string `json:"paths"` // JSON 文件路径列表
	NoRetry bool     `json:"no_retry,omitempty"`
}

// MarkUserRequest 标记用户请求
type MarkUserRequest struct {
	Timestamp *time.Time `json:"timestamp,omitempty"` // nil 表示清除标记
}

// TaskProgress 任务进度
type TaskProgress struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

// TaskResult 任务结果
type TaskResult struct {
	Downloaded int      `json:"downloaded"`
	Failed     int      `json:"failed"`
	Skipped    int      `json:"skipped"`
	Details    []string `json:"details,omitempty"`
}

// TaskResponse 任务响应
type TaskResponse struct {
	TaskID    string       `json:"task_id"`
	Type      string       `json:"type"`
	Status    string       `json:"status"` // queued, running, completed, failed, cancelled
	Progress  TaskProgress `json:"progress,omitempty"`
	Result    *TaskResult  `json:"result,omitempty"`
	Error     string       `json:"error,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	StartedAt *time.Time   `json:"started_at,omitempty"`
	EndedAt   *time.Time   `json:"ended_at,omitempty"`
}

// TasksResponse 任务列表响应
type TasksResponse struct {
	Tasks []TaskResponse `json:"tasks"`
	Total int            `json:"total"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID         uint64 `json:"id"`
	ScreenName string `json:"screen_name"`
	Name       string `json:"name"`
}

// BatchDownloadRequest 批量下载请求
type BatchDownloadRequest struct {
	Users       []string `json:"users,omitempty"`        // 用户列表
	Lists       []uint64 `json:"lists,omitempty"`        // 列表ID列表
	AutoFollow  bool     `json:"auto_follow,omitempty"`  // 自动关注
	SkipProfile bool     `json:"skip_profile,omitempty"` // 跳过Profile下载
	NoRetry     bool     `json:"no_retry,omitempty"`     // 不重试
}
