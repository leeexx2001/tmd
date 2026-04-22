package api

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusQueued    TaskStatus = "queued"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskType 任务类型
type TaskType string

const (
	TaskTypeUserDownload        TaskType = "user_download"
	TaskTypeListDownload        TaskType = "list_download"
	TaskTypeProfileDownload     TaskType = "profile_download"
	TaskTypeJsonDownload        TaskType = "json_download"
	TaskTypeMarkDownloaded      TaskType = "mark_downloaded"
	TaskTypeFollowingDownload   TaskType = "following_download"
	TaskTypeListProfileDownload TaskType = "list_profile_download"
	TaskTypeBatchDownload       TaskType = "batch_download"
)

// Task 任务定义
type Task struct {
	ID         string
	Type       TaskType
	Status     TaskStatus
	Progress   TaskProgress
	Result     *TaskResult
	Error      error
	CreatedAt  time.Time
	StartedAt  *time.Time
	EndedAt    *time.Time
	CancelFunc context.CancelFunc
	Ctx        context.Context

	// 任务特定数据
	Data interface{}
}

// TaskManager 任务管理器
type TaskManager struct {
	tasks      map[string]*Task
	mu         sync.RWMutex
	workers    int
	maxTasks   int           // 最大任务数限制
	maxTaskAge time.Duration // 任务最大存活时间
}

// NewTaskManager 创建任务管理器
func NewTaskManager(workers int) *TaskManager {
	if workers <= 0 {
		workers = 5
	}
	tm := &TaskManager{
		tasks:      make(map[string]*Task),
		workers:    workers,
		maxTasks:   1000,           // 最多保留1000个任务
		maxTaskAge: 24 * time.Hour, // 任务保留24小时
	}
	// 启动定时清理
	go tm.cleanupRoutine()
	return tm
}

// cleanupRoutine 定时清理任务
func (tm *TaskManager) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		tm.CleanupOldTasks(tm.maxTaskAge)
		tm.cleanupExceededMaxTasks()
	}
}

// cleanupExceededMaxTasks 清理超出最大数量的任务
func (tm *TaskManager) cleanupExceededMaxTasks() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if len(tm.tasks) <= tm.maxTasks {
		return
	}

	// 收集所有已完成的任务
	var completedTasks []*Task
	for _, task := range tm.tasks {
		if task.Status == TaskStatusCompleted || task.Status == TaskStatusFailed || task.Status == TaskStatusCancelled {
			completedTasks = append(completedTasks, task)
		}
	}

	// 按结束时间排序，删除最旧的任务
	if len(completedTasks) > tm.maxTasks/2 {
		sortTasksByEndTime(completedTasks)
		toDelete := len(completedTasks) - tm.maxTasks/2
		for i := 0; i < toDelete && i < len(completedTasks); i++ {
			delete(tm.tasks, completedTasks[i].ID)
		}
		log.Infof("[TaskManager] Cleaned up %d old tasks", toDelete)
	}
}

// sortTasksByEndTime 按结束时间排序（最早的在前）
func sortTasksByEndTime(tasks []*Task) {
	sort.Slice(tasks, func(i, j int) bool {
		// nil 视为最早（排前面）
		if tasks[i].EndedAt == nil && tasks[j].EndedAt != nil {
			return true
		}
		if tasks[i].EndedAt != nil && tasks[j].EndedAt == nil {
			return false
		}
		if tasks[i].EndedAt == nil && tasks[j].EndedAt == nil {
			return false
		}
		return tasks[i].EndedAt.Before(*tasks[j].EndedAt)
	})
}

// GetTaskCount 获取任务数量
func (tm *TaskManager) GetTaskCount() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.tasks)
}

// GetTaskCountByStatus 获取指定状态的任务数量
func (tm *TaskManager) GetTaskCountByStatus(status TaskStatus) int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	count := 0
	for _, task := range tm.tasks {
		if task.Status == status {
			count++
		}
	}
	return count
}

// CreateTask 创建新任务
func (tm *TaskManager) CreateTask(taskType TaskType, data interface{}) *Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	task := &Task{
		ID:         generateTaskID(),
		Type:       taskType,
		Status:     TaskStatusQueued,
		CreatedAt:  time.Now(),
		CancelFunc: cancel,
		Ctx:        ctx,
		Data:       data,
	}

	tm.tasks[task.ID] = task
	log.Infof("[TaskManager] Created task %s of type %s", task.ID, taskType)
	return task
}

// GetTask 获取任务
func (tm *TaskManager) GetTask(id string) (*Task, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	task, ok := tm.tasks[id]
	return task, ok
}

// GetAllTasks 获取所有任务
func (tm *TaskManager) GetAllTasks() []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// UpdateTaskStatus 更新任务状态
func (tm *TaskManager) UpdateTaskStatus(id string, status TaskStatus) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if task, ok := tm.tasks[id]; ok {
		task.Status = status
		now := time.Now()
		switch status {
		case TaskStatusRunning:
			task.StartedAt = &now
		case TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
			task.EndedAt = &now
		}
	}
}

// UpdateTaskProgress 更新任务进度
func (tm *TaskManager) UpdateTaskProgress(id string, progress TaskProgress) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if task, ok := tm.tasks[id]; ok {
		task.Progress = progress
	}
}

// SetTaskResult 设置任务结果
func (tm *TaskManager) SetTaskResult(id string, result *TaskResult) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if task, ok := tm.tasks[id]; ok {
		task.Result = result
	}
}

// SetTaskError 设置任务错误
func (tm *TaskManager) SetTaskError(id string, err error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if task, ok := tm.tasks[id]; ok {
		task.Error = err
		task.Status = TaskStatusFailed
		now := time.Now()
		task.EndedAt = &now
	}
}

// CancelTask 取消任务
func (tm *TaskManager) CancelTask(id string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if task, ok := tm.tasks[id]; ok {
		if task.Status == TaskStatusQueued || task.Status == TaskStatusRunning {
			task.CancelFunc()
			task.Status = TaskStatusCancelled
			now := time.Now()
			task.EndedAt = &now
			return true
		}
	}
	return false
}

// DeleteTask 删除任务（清理已完成任务）
func (tm *TaskManager) DeleteTask(id string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.tasks, id)
}

// CleanupOldTasks 清理旧任务
func (tm *TaskManager) CleanupOldTasks(maxAge time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, task := range tm.tasks {
		if task.EndedAt != nil && task.EndedAt.Before(cutoff) {
			delete(tm.tasks, id)
		}
	}
}

// generateTaskID 生成任务 ID
func generateTaskID() string {
	return fmt.Sprintf("task_%s", uuid.New().String()[:8])
}

// ToResponse 转换为响应格式
func (t *Task) ToResponse() TaskResponse {
	resp := TaskResponse{
		TaskID:    t.ID,
		Type:      string(t.Type),
		Status:    string(t.Status),
		Progress:  t.Progress,
		CreatedAt: t.CreatedAt,
		StartedAt: t.StartedAt,
		EndedAt:   t.EndedAt,
	}

	if t.Result != nil {
		resp.Result = t.Result
	}

	if t.Error != nil {
		resp.Error = t.Error.Error()
	}

	return resp
}
