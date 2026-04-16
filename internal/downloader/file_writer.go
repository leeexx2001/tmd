package downloader

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// DefaultFileWriter 默认文件写入器实现
type DefaultFileWriter struct {
	versionManager VersionManager
	locks          sync.Map // map[string]*sync.Mutex — 按文件路径的细粒度锁
}

func (fw *DefaultFileWriter) getLock(path string) *sync.Mutex {
	actual, _ := fw.locks.LoadOrStore(path, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

// NewFileWriter 创建文件写入器
func NewFileWriter(versionManager VersionManager) *DefaultFileWriter {
	return &DefaultFileWriter{
		versionManager: versionManager,
	}
}

// Write 写入文件
func (fw *DefaultFileWriter) Write(req WriteRequest) (WriteResult, error) {
	result := WriteResult{NewSize: int64(len(req.Data))}

	lock := fw.getLock(req.Path)
	lock.Lock()
	defer lock.Unlock()

	// 1. 检查是否需要跳过
	if req.Options.SkipUnchanged {
		exists, fileInfo, err := fw.fileExists(req.Path)
		if err != nil {
			return result, err
		}
		if exists {
			result.OldSize = fileInfo.Size()
			if fileInfo.Size() == result.NewSize {
				oldHash, hashErr := fw.computeFileHash(req.Path)
				if hashErr != nil {
					log.Warnf("failed to compute file hash for SkipUnchanged check: %v, path: %s", hashErr, req.Path)
					return result, hashErr
				}
				newHash := fw.computeDataHash(req.Data)
				if oldHash == newHash {
					result.Skipped = true
					result.Success = true
					return result, nil
				}
			}
		}
	}

	// 2. 创建版本备份（如果需要）
	if req.Options.CreateVersion && fw.versionManager != nil {
		if _, err := os.Stat(req.Path); err == nil {
			_, err := fw.versionManager.CreateVersion(req.Path)
			if err != nil {
				return result, err
			}
		}
	}

	// 3. 原子写入
	if err := fw.atomicWrite(req.Path, req.Data); err != nil {
		return result, err
	}

	// 4. 设置修改时间
	if req.Options.ModTime != nil {
		if err := os.Chtimes(req.Path, time.Time{}, *req.Options.ModTime); err != nil {
			log.Warnf("failed to set modification time for %s: %v", req.Path, err)
		}
	}

	result.Success = true
	return result, nil
}

// fileExists 检查文件是否存在
func (fw *DefaultFileWriter) fileExists(path string) (bool, os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, info, nil
}

// computeFileHash 计算文件 Hash
func (fw *DefaultFileWriter) computeFileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return fw.computeDataHash(data), nil
}

// computeDataHash 计算数据 Hash
func (fw *DefaultFileWriter) computeDataHash(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// atomicWrite 原子写入
func (fw *DefaultFileWriter) atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(dir, ".tmp_*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()

	defer os.Remove(tempPath)

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, path)
}
