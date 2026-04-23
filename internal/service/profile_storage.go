package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/unkmonster/tmd/internal/downloader"
)

const (
	profileDirName    = ".loongtweet"
	profileSubDirName = ".profile"
	versionsDirName   = ".versions"
)

// profileStorageManager Profile 存储管理器（包内私有）
type profileStorageManager struct {
	usersBasePath  string
	versionManager downloader.VersionManager
}

// newProfileStorageManager 创建存储管理器（包内私有）
func newProfileStorageManager(usersBasePath string) (*profileStorageManager, error) {
	if usersBasePath == "" {
		return nil, fmt.Errorf("usersBasePath cannot be empty")
	}

	if err := os.MkdirAll(usersBasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create users base directory: %w", err)
	}

	return &profileStorageManager{usersBasePath: usersBasePath}, nil
}

func (fsm *profileStorageManager) setVersionManager(vm downloader.VersionManager) {
	fsm.versionManager = vm
}

func (fsm *profileStorageManager) getUserProfilePath(userTitle string) string {
	return filepath.Join(fsm.usersBasePath, userTitle, profileDirName, profileSubDirName)
}

func (fsm *profileStorageManager) ensureDirectory(userTitle string) (string, error) {
	userDir := filepath.Join(fsm.usersBasePath, userTitle)
	return ensureProfileDirs(userDir)
}

// ensureProfileDirs 确保 Profile 目录存在（从原 downloader.go 移动）
func ensureProfileDirs(userDir string) (string, error) {
	profileDir := filepath.Join(userDir, profileDirName, profileSubDirName)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create profile dir: %w", err)
	}
	versionsDir := filepath.Join(profileDir, versionsDirName)
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create versions dir: %w", err)
	}
	return profileDir, nil
}

func (fsm *profileStorageManager) getFilePath(userTitle string, fileType FileType) string {
	profilePath := fsm.getUserProfilePath(userTitle)

	switch fileType {
	case FileTypeAvatar:
		return filepath.Join(profilePath, "avatar.jpg")
	case FileTypeBanner:
		return filepath.Join(profilePath, "banner.jpg")
	case FileTypeDescription:
		return filepath.Join(profilePath, "description.txt")
	case FileTypeProfile:
		return filepath.Join(profilePath, "profile.json")
	default:
		return filepath.Join(profilePath, string(fileType))
	}
}

func (fsm *profileStorageManager) getFilePathWithExt(userTitle string, fileType FileType, ext string) string {
	profilePath := fsm.getUserProfilePath(userTitle)

	switch fileType {
	case FileTypeAvatar:
		filename := "avatar" + ext
		return filepath.Join(profilePath, filename)
	case FileTypeBanner:
		filename := "banner" + ext
		return filepath.Join(profilePath, filename)
	default:
		return fsm.getFilePath(userTitle, fileType)
	}
}
