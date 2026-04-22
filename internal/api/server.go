package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/config"
	"github.com/unkmonster/tmd/internal/downloader"
	"github.com/unkmonster/tmd/internal/downloading"
	"github.com/unkmonster/tmd/internal/twitter"
)

// Server API Server
type Server struct {
	httpServer        *http.Server
	taskManager       *TaskManager
	config            *config.Config
	db                *sqlx.DB
	client            *resty.Client
	additionalClients []*resty.Client
	dwn               downloader.Downloader
	fileWriter        downloader.FileWriter
	versionManager    downloader.VersionManager

	// 存储路径
	storePath *storePathHelper

	// 错误持久化
	dumper   *downloading.TweetDumper
	dumperMu sync.Mutex
}

type storePathHelper struct {
	root   string
	users  string
	data   string
	db     string
	errorj string
}

// ServerOptions Server 配置选项
type ServerOptions struct {
	Port              string
	Config            *config.Config
	DB                *sqlx.DB
	Client            *resty.Client
	AdditionalClients []*resty.Client
	Downloader        downloader.Downloader
	FileWriter        downloader.FileWriter
	VersionManager    downloader.VersionManager
	StoreRoot         string
	StoreUsers        string
	StoreData         string
	StoreDB           string
	StoreErrorJ       string
}

// NewServer 创建 API Server
func NewServer(opts *ServerOptions) (*Server, error) {
	if opts.Port == "" {
		opts.Port = "25556"
	}

	mux := http.NewServeMux()

	// 初始化 Dumper
	dumper := downloading.NewDumper()
	if err := dumper.Load(opts.StoreErrorJ); err != nil {
		log.Warnf("[API Server] Failed to load errors.json: %v", err)
	} else if dumper.Count() > 0 {
		log.Infof("[API Server] Loaded %d failed tweets from errors.json", dumper.Count())
	}

	server := &Server{
		taskManager:       NewTaskManager(5),
		config:            opts.Config,
		db:                opts.DB,
		client:            opts.Client,
		additionalClients: opts.AdditionalClients,
		dwn:               opts.Downloader,
		fileWriter:        opts.FileWriter,
		versionManager:    opts.VersionManager,
		storePath: &storePathHelper{
			root:   opts.StoreRoot,
			users:  opts.StoreUsers,
			data:   opts.StoreData,
			db:     opts.StoreDB,
			errorj: opts.StoreErrorJ,
		},
		dumper: dumper,
	}

	// 注册路由
	server.registerRoutes(mux)

	server.httpServer = &http.Server{
		Addr:         ":" + opts.Port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server, nil
}

// registerRoutes 注册路由
func (s *Server) registerRoutes(mux *http.ServeMux) {
	// 健康检查
	mux.HandleFunc("/api/v1/health", enableCORS(s.handleHealth))

	// 用户相关
	mux.HandleFunc("/api/v1/users/", enableCORS(s.handleUsers))

	// 列表相关
	mux.HandleFunc("/api/v1/lists/", enableCORS(s.handleLists))

	// JSON 下载
	mux.HandleFunc("/api/v1/json/download", enableCORS(s.handleJsonDownload))

	// 任务管理
	mux.HandleFunc("/api/v1/tasks", enableCORS(s.handleTasks))
	mux.HandleFunc("/api/v1/tasks/", enableCORS(s.handleTaskDetail))

	// 批量操作
	mux.HandleFunc("/api/v1/batch/download", enableCORS(s.handleBatchDownload))

	// 重试接口
	mux.HandleFunc("/api/v1/retry", enableCORS(s.handleRetry))
}

// Start 启动 Server
func (s *Server) Start() error {
	log.Infof("[API Server] Starting on port %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop 停止 Server
func (s *Server) Stop(ctx context.Context) error {
	log.Info("[API Server] Shutting down...")

	// 保存错误到文件
	s.dumperMu.Lock()
	if s.dumper.Count() > 0 {
		if err := s.dumper.Dump(s.storePath.errorj); err != nil {
			log.Errorf("[API Server] Failed to save errors.json: %v", err)
		} else {
			log.Infof("[API Server] Saved %d failed tweets to errors.json", s.dumper.Count())
		}
	}
	s.dumperMu.Unlock()

	return s.httpServer.Shutdown(ctx)
}

// handleHealth 健康检查处理器
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: HealthResponse{
			Status:    "ok",
			Version:   "2.0.0",
			Timestamp: time.Now(),
		},
	})
}

// handleUsers 处理用户相关请求
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "Invalid user path")
		return
	}

	screenName := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	// 处理 /users/{screen_name}/following/download 路径
	if len(parts) > 2 && parts[1] == "following" && parts[2] == "download" {
		action = "following_download"
	}

	switch r.Method {
	case http.MethodPost:
		switch action {
		case "download":
			s.handleUserDownload(w, r, screenName)
		case "profile":
			s.handleUserProfile(w, r, screenName)
		case "mark":
			s.handleUserMark(w, r, screenName)
		case "following_download":
			s.handleUserFollowingDownload(w, r, screenName)
		default:
			writeError(w, http.StatusNotFound, "Unknown action")
		}
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleLists 处理列表相关请求
func (s *Server) handleLists(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/lists/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "Invalid list path")
		return
	}

	listID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch r.Method {
	case http.MethodPost:
		switch action {
		case "download":
			s.handleListDownload(w, r, listID)
		case "profile":
			s.handleListProfile(w, r, listID)
		default:
			writeError(w, http.StatusNotFound, "Unknown action")
		}
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleTasks 处理任务列表
func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	tasks := s.taskManager.GetAllTasks()
	resp := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		resp[i] = task.ToResponse()
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: TasksResponse{
			Tasks: resp,
			Total: len(resp),
		},
	})
}

// handleTaskDetail 处理单个任务详情
func (s *Server) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
	taskID := strings.Split(path, "/")[0]

	task, ok := s.taskManager.GetTask(taskID)
	if !ok {
		writeError(w, http.StatusNotFound, "Task not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Data:    task.ToResponse(),
		})
	case http.MethodPost:
		// 检查是否是取消操作
		if strings.HasSuffix(path, "/cancel") {
			if s.taskManager.CancelTask(taskID) {
				writeJSON(w, http.StatusOK, Response{
					Success: true,
					Data:    map[string]string{"message": "Task cancelled"},
				})
			} else {
				writeError(w, http.StatusBadRequest, "Task cannot be cancelled")
			}
		} else {
			writeError(w, http.StatusNotFound, "Unknown action")
		}
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleJsonDownload 处理 JSON 下载
func (s *Server) handleJsonDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req JsonDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Paths) == 0 {
		writeError(w, http.StatusBadRequest, "No JSON paths provided")
		return
	}

	task := s.taskManager.CreateTask(TaskTypeJsonDownload, &JsonDownloadTaskData{
		Paths:   req.Paths,
		NoRetry: req.NoRetry,
	})

	go s.executeJsonDownload(task)

	writeJSON(w, http.StatusAccepted, Response{
		Success: true,
		Data: map[string]interface{}{
			"task_id": task.ID,
			"status":  string(task.Status),
			"message": "JSON download task queued",
		},
	})
}

// handleBatchDownload 处理批量下载请求
func (s *Server) handleBatchDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req BatchDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 验证至少有一个下载目标
	if len(req.Users) == 0 && len(req.Lists) == 0 {
		writeError(w, http.StatusBadRequest, "No download targets provided")
		return
	}

	// 创建批量任务
	task := s.taskManager.CreateTask(TaskTypeBatchDownload, &BatchDownloadTaskData{
		Users:       req.Users,
		Lists:       req.Lists,
		AutoFollow:  req.AutoFollow,
		SkipProfile: req.SkipProfile,
		NoRetry:     req.NoRetry,
	})

	// 异步执行批量任务
	go s.executeBatchDownload(task)

	writeJSON(w, http.StatusAccepted, Response{
		Success: true,
		Data: map[string]interface{}{
			"task_id":    task.ID,
			"status":     string(task.Status),
			"user_count": len(req.Users),
			"list_count": len(req.Lists),
			"message":    "Batch download task queued",
		},
	})
}

// 辅助函数
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, Response{
		Success: false,
		Error:   message,
	})
}

// enableCORS 启用跨域支持
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("[API] %s %s", r.Method, r.URL.Path)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		next(w, r)
	}
}

// PushFailedTweets 添加失败推文到 Dumper
func (s *Server) PushFailedTweets(eid int, tweets ...*twitter.Tweet) {
	s.dumperMu.Lock()
	defer s.dumperMu.Unlock()
	s.dumper.Push(eid, tweets...)
}

// GetFailedTweetsCount 获取失败推文数量
func (s *Server) GetFailedTweetsCount() int {
	s.dumperMu.Lock()
	defer s.dumperMu.Unlock()
	return s.dumper.Count()
}

// RetryFailedTweets 重试失败的推文
func (s *Server) RetryFailedTweets(ctx context.Context) error {
	s.dumperMu.Lock()
	defer s.dumperMu.Unlock()

	if s.dumper.Count() == 0 {
		return nil
	}

	log.Infoln("[API Server] Starting to retry failed tweets")
	if err := downloading.RetryFailedTweets(ctx, s.dumper, s.db, s.client, s.dwn, s.fileWriter); err != nil {
		return err
	}

	log.Infof("[API Server] Retry completed, %d tweets still failed", s.dumper.Count())
	return nil
}

// SaveDumper 立即保存 Dumper 到文件
func (s *Server) SaveDumper() error {
	s.dumperMu.Lock()
	defer s.dumperMu.Unlock()

	if s.dumper.Count() > 0 {
		return s.dumper.Dump(s.storePath.errorj)
	}
	return nil
}

// handleRetry 处理重试请求
func (s *Server) handleRetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req RetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = RetryRequest{}
	}

	beforeCount := s.GetFailedTweetsCount()

	if beforeCount == 0 {
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Data: RetryResponse{
				BeforeCount: 0,
				AfterCount:  0,
				Retried:     0,
				Message:     "No failed tweets to retry",
			},
		})
		return
	}

	// 创建重试任务
	task := s.taskManager.CreateTask(TaskTypeRetry, &RetryTaskData{
		NoRetry: req.NoRetry,
	})

	// 异步执行
	go s.executeRetry(task)

	writeJSON(w, http.StatusAccepted, Response{
		Success: true,
		Data: map[string]interface{}{
			"task_id":      task.ID,
			"status":       string(task.Status),
			"failed_count": beforeCount,
			"message":      "Retry task queued",
		},
	})
}
