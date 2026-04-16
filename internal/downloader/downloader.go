package downloader

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
)

// DefaultDownloader 默认下载器实现
type DefaultDownloader struct {
	fileWriter FileWriter
	logger     log.FieldLogger
}

// NewDownloader 创建下载器
func NewDownloader(fileWriter FileWriter) *DefaultDownloader {
	return &DefaultDownloader{
		fileWriter: fileWriter,
		logger:     log.StandardLogger(),
	}
}

// Download 下载单个文件
func (d *DefaultDownloader) Download(req DownloadRequest) (*DownloadResult, error) {
	result := &DownloadResult{}

	// 回调：下载前
	if req.Options.OnBeforeDownload != nil {
		req.Options.OnBeforeDownload(&req)
	}

	// 构建请求
	r := req.Client.R().SetContext(req.Context)
	for k, v := range req.Options.QueryParams {
		r = r.SetQueryParam(k, v)
	}

	// 执行下载
	resp, err := r.Get(req.URL)
	if err != nil {
		result.Error = err
		return result, err
	}

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		err := fmt.Errorf("HTTP %d: %s", resp.StatusCode(), req.URL)
		result.Error = err
		d.logger.WithFields(log.Fields{
			"url":         req.URL,
			"status_code": resp.StatusCode(),
		}).Warn("download failed with non-2xx status")
		return result, err
	}

	// 写入文件
	writeReq := WriteRequest{
		Path: req.Destination,
		Data: resp.Body(),
		Options: WriteOptions{
			CreateVersion: req.Options.CreateVersion,
			SkipUnchanged: req.Options.SkipUnchanged,
			ModTime:       req.Options.SetModTime,
		},
	}
	writeResult, err := d.fileWriter.Write(writeReq)
	if err != nil {
		result.Error = err
		result.Success = false
		return result, err
	}

	result.Success = writeResult.Success
	result.Skipped = writeResult.Skipped
	result.FilePath = req.Destination
	result.FileSize = writeResult.NewSize
	result.OldSize = writeResult.OldSize

	// 回调：下载后
	if req.Options.OnAfterDownload != nil {
		req.Options.OnAfterDownload(result)
	}

	return result, nil
}

// BatchDownload 批量下载（并发，支持上下文取消）
func (d *DefaultDownloader) BatchDownload(ctx context.Context, reqs []DownloadRequest) ([]*DownloadResult, error) {
	results := make([]*DownloadResult, len(reqs))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, req := range reqs {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		wg.Add(1)
		go func(index int, request DownloadRequest) {
			defer wg.Done()

			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				mu.Lock()
				results[index] = &DownloadResult{Error: ctx.Err()}
				mu.Unlock()
				return
			default:
			}

			result, err := d.Download(request)

			mu.Lock()
			results[index] = result
			if err != nil && firstErr == nil {
				firstErr = err
			}
			mu.Unlock()
		}(i, req)
	}

	wg.Wait()
	return results, firstErr
}
