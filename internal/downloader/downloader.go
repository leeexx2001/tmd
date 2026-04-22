package downloader

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// 流式下载阈值：10MB
const streamThreshold = 10 * 1024 * 1024

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

// Download 下载单个文件（智能选择：大文件流式，小文件 Buffer）
func (d *DefaultDownloader) Download(req DownloadRequest) (*DownloadResult, error) {
	// 1. 获取文件大小（HEAD 请求）
	contentLength, err := d.getContentLength(req)
	if err != nil {
		// HEAD 失败，回退到 Buffer 模式
		d.logger.WithFields(log.Fields{
			"url":   req.URL,
			"error": err,
		}).Debug("HEAD request failed, fallback to buffer mode")
		return d.downloadBuffer(req)
	}

	// 2. 根据大小选择策略
	if contentLength > streamThreshold {
		// 大文件：流式下载
		d.logger.WithFields(log.Fields{
			"url":  req.URL,
			"size": contentLength,
		}).Debug("using stream mode for large file")
		return d.downloadStream(req, contentLength)
	}

	// 小文件：Buffer 下载（支持 SkipUnchanged）
	d.logger.WithFields(log.Fields{
		"url":  req.URL,
		"size": contentLength,
	}).Debug("using buffer mode for small file")
	return d.downloadBuffer(req)
}

// getContentLength 通过 HEAD 请求获取文件大小
func (d *DefaultDownloader) getContentLength(req DownloadRequest) (int64, error) {
	headReq := req.Client.R().
		SetContext(req.Context).
		SetDoNotParseResponse(true)

	for k, v := range req.Options.QueryParams {
		headReq = headReq.SetQueryParam(k, v)
	}

	resp, err := headReq.Head(req.URL)
	if err != nil {
		return 0, err
	}

	// 确保关闭响应体，防止连接泄漏
	if resp.RawBody() != nil {
		resp.RawBody().Close()
	}

	if resp.RawResponse == nil {
		return 0, fmt.Errorf("no response")
	}

	contentLength := resp.RawResponse.ContentLength
	if contentLength <= 0 {
		return 0, fmt.Errorf("unknown content length: %d", contentLength)
	}

	return contentLength, nil
}

// downloadBuffer 原有 Buffer 模式（小文件）
func (d *DefaultDownloader) downloadBuffer(req DownloadRequest) (*DownloadResult, error) {
	result := &DownloadResult{}

	r := req.Client.R().SetContext(req.Context)
	for k, v := range req.Options.QueryParams {
		r = r.SetQueryParam(k, v)
	}

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

	return result, nil
}

// downloadStream 流式下载（大文件）
func (d *DefaultDownloader) downloadStream(req DownloadRequest, contentLength int64) (*DownloadResult, error) {
	result := &DownloadResult{}

	r := req.Client.R().
		SetContext(req.Context).
		SetDoNotParseResponse(true) // 关键：不自动解析响应体

	for k, v := range req.Options.QueryParams {
		r = r.SetQueryParam(k, v)
	}

	resp, err := r.Get(req.URL)
	if err != nil {
		result.Error = err
		return result, err
	}

	// 检查响应体是否存在
	if resp.RawBody() == nil {
		result.Error = fmt.Errorf("no response body")
		return result, result.Error
	}
	defer resp.RawBody().Close()

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		err := fmt.Errorf("HTTP %d: %s", resp.StatusCode(), req.URL)
		result.Error = err
		d.logger.WithFields(log.Fields{
			"url":         req.URL,
			"status_code": resp.StatusCode(),
		}).Warn("download failed with non-2xx status")
		return result, err
	}

	writeReq := WriteRequest{
		Path:   req.Destination,
		Reader: resp.RawBody(),
		Size:   contentLength,
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

	return result, nil
}
