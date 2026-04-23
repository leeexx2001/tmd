package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/downloading"
	"github.com/unkmonster/tmd/internal/twitter"
)

// MarkService 标记服务
type MarkService struct {
	client *resty.Client
	db     *sqlx.DB
}

// NewMarkService 创建标记服务
func NewMarkService(client *resty.Client, db *sqlx.DB) *MarkService {
	return &MarkService{
		client: client,
		db:     db,
	}
}

// MarkDownloadedRequest 标记已下载请求
type MarkDownloadedRequest struct {
	Users     []*twitter.User
	Timestamp *time.Time
}

// ExecuteMarkDownloaded 执行标记已下载
func (s *MarkService) ExecuteMarkDownloaded(ctx context.Context, req *MarkDownloadedRequest) error {
	var markTimeStr string
	if req.Timestamp == nil {
		markTimeStr = ""
	} else {
		markTimeStr = req.Timestamp.Format("2006-01-02T15:04:05")
	}

	results, err := downloading.MarkUsersAsDownloaded(ctx, s.client, s.db, nil, req.Users, "", markTimeStr)
	if err != nil {
		return fmt.Errorf("failed to mark users as downloaded: %w", err)
	}

	if len(results) > 0 {
		fmt.Println("\n=== MARK_DOWNLOADED_RESULTS ===")
		for _, r := range results {
			status := "OK"
			if !r.Success {
				status = "FAIL"
			}
			fmt.Printf("ENTITY_ID:%d|USER_ID:%d|SCREEN_NAME:%s|STATUS:%s\n", r.EntityID, r.UserID, r.ScreenName, status)
		}
		fmt.Println("=== END_RESULTS ===")
	}

	return nil
}

// MarkUsersRequest 标记用户请求
type MarkUsersRequest struct {
	Users []*twitter.User
}

// ExecuteMarkUsers 执行标记用户
func (s *MarkService) ExecuteMarkUsers(ctx context.Context, req *MarkUsersRequest) error {
	now := time.Now()
	results, err := downloading.MarkUsersAsDownloaded(ctx, s.client, s.db, nil, req.Users, "", now.Format("2006-01-02T15:04:05"))
	if err != nil {
		return fmt.Errorf("failed to mark users: %w", err)
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	log.Infof("Marked %d/%d users as downloaded", successCount, len(results))

	return nil
}
