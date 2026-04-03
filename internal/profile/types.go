package profile

import (
	"context"
	"fmt"
	"time"
)

// ProfileInfo 用户资料信息
type ProfileInfo struct {
	ID          uint64 `json:"ID"`
	Name        string `json:"Name"`
	ScreenName  string `json:"ScreenName"`
	Description string `json:"-"` // 用户简介（不保存到JSON，单独保存为description.txt）
	AvatarURL   string `json:"-"`
	BannerURL   string `json:"-"`
	URL         string `json:"URL"`
	Location    string `json:"Location"`
	Verified    bool   `json:"Verified"`
	Protected   bool   `json:"Protected"`
	CreatedAt   string `json:"CreatedAt"`
}

// FileType 文件类型
type FileType string

const (
	FileTypeAvatar      FileType = "avatar"
	FileTypeBanner      FileType = "banner"
	FileTypeDescription FileType = "description"
	FileTypeProfile     FileType = "profile"
)

// FileStatus 文件处理状态
type FileStatus int

const (
	StatusFailed FileStatus = iota
	StatusDownloaded
	StatusSkipped
	StatusVersioned
)

func (s FileStatus) String() string {
	switch s {
	case StatusFailed:
		return "failed"
	case StatusDownloaded:
		return "downloaded"
	case StatusSkipped:
		return "skipped"
	case StatusVersioned:
		return "versioned"
	default:
		return "unknown"
	}
}

// FileResult 单个文件下载结果
type FileResult struct {
	FileType    FileType
	FilePath    string
	Status      FileStatus
	OldSize     int64
	NewSize     int64
	VersionPath string
	Error       error
}

// DownloadResult 下载结果
type DownloadResult struct {
	ScreenName   string
	Success      bool
	Files        []FileResult
	Error        error
	DownloadTime time.Duration
	Profile      *ProfileInfo
}

// Config 下载器配置
type Config struct {
	// 存储根目录，默认为 ~/.loongtweet/.profile
	RootPath string
	// 是否启用版本管理
	EnableVersioning bool
	// 下载间隔（批量下载时使用）
	DownloadInterval time.Duration
	// 是否跳过已存在的未变更文件
	SkipUnchanged bool
	// 头像图片质量 (如 "400x400", "200x200")
	AvatarQuality string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		RootPath:         "",
		EnableVersioning: true,
		DownloadInterval: 0,
		SkipUnchanged:    true,
		AvatarQuality:    "400x400",
	}
}

// Downloader 下载器接口
type Downloader interface {
	Download(ctx context.Context, screenName string) (*DownloadResult, error)
	DownloadMultiple(ctx context.Context, screenNames []string) []*DownloadResult
}

// StorageManager 存储管理器接口
type StorageManager interface {
	EnsureDirectory(screenName string) (string, error)
	GetFilePath(screenName string, fileType FileType) string
	GetVersionPath(screenName string, fileType FileType, timestamp time.Time) string
	FileExists(path string) (bool, FileInfo, error)
}

// FileInfo 文件信息
type FileInfo struct {
	Path    string
	Size    int64
	ModTime time.Time
}

// Fetcher 远程数据获取器接口
type Fetcher interface {
	FetchProfile(ctx context.Context, screenName string) (*ProfileInfo, error)
	FetchAvatar(ctx context.Context, url string) ([]byte, error)
	FetchBanner(ctx context.Context, url string) ([]byte, string, error)
}

// ProfileError 自定义错误类型
type ProfileError struct {
	Op   string
	User string
	Err  error
}

func (e *ProfileError) Error() string {
	if e.User != "" {
		return fmt.Sprintf("profile %s failed for user %s: %v", e.Op, e.User, e.Err)
	}
	return fmt.Sprintf("profile %s failed: %v", e.Op, e.Err)
}

func (e *ProfileError) Unwrap() error {
	return e.Err
}

// IsNotFound 检查是否为"用户不存在"错误
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	// 这里可以根据实际错误类型进行判断
	return false
}
