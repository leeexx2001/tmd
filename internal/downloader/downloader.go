package downloader

import (
	"fmt"

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

	return result, nil
}
