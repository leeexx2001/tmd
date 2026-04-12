package profile

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/unkmonster/tmd/internal/utils"
)

type FileStorageManager struct {
	usersBasePath string
}

func getFileExtension(fileType FileType) string {
	switch fileType {
	case FileTypeDescription:
		return ".txt"
	case FileTypeProfile:
		return ".json"
	default:
		return ".jpg"
	}
}

func NewFileStorageManager(usersBasePath string) (*FileStorageManager, error) {
	if usersBasePath == "" {
		return nil, fmt.Errorf("usersBasePath cannot be empty")
	}

	if err := os.MkdirAll(usersBasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create users base directory: %w", err)
	}

	return &FileStorageManager{usersBasePath: usersBasePath}, nil
}

func (fsm *FileStorageManager) getUserProfilePath(userTitle string) string {
	// userTitle 已经在外部清理过，直接使用
	return filepath.Join(fsm.usersBasePath, userTitle, profileDirName, profileSubDirName)
}

func (fsm *FileStorageManager) EnsureDirectory(userTitle string) (string, error) {
	profilePath := fsm.getUserProfilePath(userTitle)

	if err := os.MkdirAll(profilePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create user profile directory: %w", err)
	}

	versionsDir := filepath.Join(profilePath, versionsDirName)
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create versions directory: %w", err)
	}

	return profilePath, nil
}

func (fsm *FileStorageManager) GetFilePath(userTitle string, fileType FileType) string {
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

func (fsm *FileStorageManager) GetFilePathWithExt(userTitle string, fileType FileType, ext string) string {
	profilePath := fsm.getUserProfilePath(userTitle)

	switch fileType {
	case FileTypeAvatar:
		filename := "avatar" + ext
		return filepath.Join(profilePath, filename)
	case FileTypeBanner:
		filename := "banner" + ext
		return filepath.Join(profilePath, filename)
	default:
		return fsm.GetFilePath(userTitle, fileType)
	}
}

func (fsm *FileStorageManager) GetVersionPath(userTitle string, fileType FileType, timestamp time.Time) string {
	ext := getFileExtension(fileType)
	return fsm.GetVersionPathWithExt(userTitle, fileType, timestamp, ext)
}

func (fsm *FileStorageManager) GetVersionPathWithExt(userTitle string, fileType FileType, timestamp time.Time, ext string) string {
	profilePath := fsm.getUserProfilePath(userTitle)
	versionsDir := filepath.Join(profilePath, versionsDirName)

	timestampStr := timestamp.Format("20060102_150405")

	switch fileType {
	case FileTypeAvatar:
		return filepath.Join(versionsDir, fmt.Sprintf("avatar_%s%s", timestampStr, ext))
	case FileTypeBanner:
		return filepath.Join(versionsDir, fmt.Sprintf("banner_%s%s", timestampStr, ext))
	case FileTypeDescription:
		return filepath.Join(versionsDir, fmt.Sprintf("description_%s%s", timestampStr, ext))
	case FileTypeProfile:
		return filepath.Join(versionsDir, fmt.Sprintf("profile_%s%s", timestampStr, ext))
	default:
		return filepath.Join(versionsDir, fmt.Sprintf("%s_%s%s", fileType, timestampStr, ext))
	}
}

func (fsm *FileStorageManager) FileExists(path string) (bool, FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, FileInfo{}, nil
		}
		return false, FileInfo{}, err
	}

	return true, FileInfo{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

func (fsm *FileStorageManager) CreateVersion(userTitle string, fileType FileType, sourcePath string) (string, error) {
	ext := filepath.Ext(sourcePath)
	versionPath := fsm.GetVersionPathWithExt(userTitle, fileType, time.Now(), ext)

	if err := utils.CopyFile(sourcePath, versionPath); err != nil {
		return "", fmt.Errorf("failed to create version backup: %w", err)
	}

	return versionPath, nil
}

// AtomicWrite 原子写入文件
// Deprecated: 使用 github.com/unkmonster/tmd/internal/downloader.FileWriter 替代
func (fsm *FileStorageManager) AtomicWrite(filePath string, data []byte) error {
	dir := filepath.Dir(filePath)
	tempFile, err := os.CreateTemp(dir, ".tmp_*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	defer os.Remove(tempPath)

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// ComputeFileHash 计算文件 Hash
// Deprecated: 使用 github.com/unkmonster/tmd/internal/downloader.FileWriter 替代
func (fsm *FileStorageManager) ComputeFileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:]), nil
}

// ComputeDataHash 计算数据 Hash
// Deprecated: 使用 github.com/unkmonster/tmd/internal/downloader.FileWriter 替代
func ComputeDataHash(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}
