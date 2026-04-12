package downloader

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultVersionManager 默认版本管理器实现
type DefaultVersionManager struct {
	versionsDirName string
}

// NewVersionManager 创建版本管理器
func NewVersionManager(versionsDirName string) *DefaultVersionManager {
	if versionsDirName == "" {
		versionsDirName = ".versions"
	}
	return &DefaultVersionManager{
		versionsDirName: versionsDirName,
	}
}

// CreateVersion 创建版本备份
func (vm *DefaultVersionManager) CreateVersion(sourcePath string) (string, error) {
	dir := filepath.Dir(sourcePath)
	filename := filepath.Base(sourcePath)
	ext := filepath.Ext(filename)
	stem := filename[:len(filename)-len(ext)]

	versionsDir := filepath.Join(dir, vm.versionsDirName)
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102_150405")
	versionPath := filepath.Join(versionsDir, fmt.Sprintf("%s_%s%s", stem, timestamp, ext))

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(versionPath, data, 0644); err != nil {
		return "", err
	}

	return versionPath, nil
}
