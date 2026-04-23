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
)

// JsonService JSON 下载服务
type JsonService struct {
	client            *resty.Client
	additionalClients []*resty.Client
	db                *sqlx.DB
	conf              *config.Config
}

// NewJsonService 创建 JSON 服务
func NewJsonService(client *resty.Client, additionalClients []*resty.Client, db *sqlx.DB, conf *config.Config) *JsonService {
	return &JsonService{
		client:            client,
		additionalClients: additionalClients,
		db:                db,
		conf:              conf,
	}
}

// DownloadJsonRequest JSON 下载请求
type DownloadJsonRequest struct {
	Paths   []string
	NoRetry bool
}

// ExecuteDownloadJson 执行 JSON 下载
func (s *JsonService) ExecuteDownloadJson(ctx context.Context, req *DownloadJsonRequest) error {
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

	log.Infoln("Starting JSON download from", len(req.Paths), "files")
	results := downloading.DownloadJsonDir(ctx, s.client, pathHelper.Root, dwn, fileWriter, req.Paths...)

	var successCount, failCount int
	for _, r := range results {
		if r.Success {
			successCount++
			log.Infof("✓ %s: %d tweets processed in %v", r.Path, r.TweetCount, r.Duration)
		} else {
			failCount++
			log.Errorf("✗ %s: %v", r.Path, r.Error)
		}
	}
	log.Infof("JSON download completed: %d success, %d failed", successCount, failCount)

	if !req.NoRetry {
		downloading.RetryFailedTweets(ctx, dumper, s.db, s.client, dwn, fileWriter)
	}

	return nil
}
