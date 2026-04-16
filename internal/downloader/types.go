package downloader

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
)

type DownloadRequest struct {
	Context     context.Context
	Client      *resty.Client
	URL         string
	Destination string
	Options     DownloadOptions
}

type DownloadOptions struct {
	QueryParams      map[string]string
	SkipUnchanged    bool
	CreateVersion    bool
	SetModTime       *time.Time
	OnBeforeDownload func(req *DownloadRequest)
	OnAfterDownload  func(result *DownloadResult)
}

type DownloadResult struct {
	Success  bool
	Skipped  bool
	FilePath string
	FileSize int64
	OldSize  int64
	Error    error
}

type WriteRequest struct {
	Path    string
	Data    []byte
	Options WriteOptions
}

type WriteOptions struct {
	CreateVersion bool
	SkipUnchanged bool
	ModTime       *time.Time
}

type WriteResult struct {
	Success bool
	Skipped bool
	OldSize int64
	NewSize int64
}

type Downloader interface {
	Download(req DownloadRequest) (*DownloadResult, error)
	BatchDownload(ctx context.Context, reqs []DownloadRequest) ([]*DownloadResult, error)
}

type FileWriter interface {
	Write(req WriteRequest) (WriteResult, error)
}

type VersionManager interface {
	CreateVersion(sourcePath string) (string, error)
}
