package path

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unkmonster/tmd/internal/utils"
)

func TestNewStorePath(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "path_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sp, err := NewStorePath(tempDir)
	if err != nil {
		t.Errorf("NewStorePath() error = %v", err)
		return
	}

	if sp == nil {
		t.Error("NewStorePath() returned nil")
		return
	}

	// 验证路径设置
	if sp.Root != tempDir {
		t.Errorf("Root = %v, want %v", sp.Root, tempDir)
	}

	expectedUsers := filepath.Join(tempDir, "users")
	if sp.Users != expectedUsers {
		t.Errorf("Users = %v, want %v", sp.Users, expectedUsers)
	}

	expectedData := filepath.Join(tempDir, ".data")
	if sp.Data != expectedData {
		t.Errorf("Data = %v, want %v", sp.Data, expectedData)
	}

	expectedDB := filepath.Join(expectedData, "foo.db")
	if sp.DB != expectedDB {
		t.Errorf("DB = %v, want %v", sp.DB, expectedDB)
	}

	expectedErrorJ := filepath.Join(expectedData, "errors.json")
	if sp.ErrorJ != expectedErrorJ {
		t.Errorf("ErrorJ = %v, want %v", sp.ErrorJ, expectedErrorJ)
	}
}

func TestNewStorePath_CreatesDirectories(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "path_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	rootPath := filepath.Join(tempDir, "store")

	_, err = NewStorePath(rootPath)
	if err != nil {
		t.Errorf("NewStorePath() error = %v", err)
		return
	}

	// 验证目录被创建
	paths := []string{
		rootPath,
		filepath.Join(rootPath, "users"),
		filepath.Join(rootPath, ".data"),
	}

	for _, p := range paths {
		exists, err := utils.PathExists(p)
		if err != nil {
			t.Errorf("PathExists(%s) error = %v", p, err)
			continue
		}
		if !exists {
			t.Errorf("Directory %s was not created", p)
		}
	}
}

func TestNewStorePath_ExistingDir(t *testing.T) {
	// 使用已存在的目录
	tempDir, err := os.MkdirTemp("", "path_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 预先创建子目录
	os.MkdirAll(filepath.Join(tempDir, "users"), 0755)
	os.MkdirAll(filepath.Join(tempDir, ".data"), 0755)

	sp, err := NewStorePath(tempDir)
	if err != nil {
		t.Errorf("NewStorePath() error = %v", err)
		return
	}

	if sp.Root != tempDir {
		t.Errorf("Root = %v, want %v", sp.Root, tempDir)
	}
}

func TestStorePath_Struct(t *testing.T) {
	// 测试结构体字段
	sp := &StorePath{
		Root:   "/root",
		Users:  "/root/users",
		Data:   "/root/.data",
		DB:     "/root/.data/foo.db",
		ErrorJ: "/root/.data/errors.json",
	}

	if sp.Root != "/root" {
		t.Errorf("Root = %v, want /root", sp.Root)
	}
	if sp.Users != "/root/users" {
		t.Errorf("Users = %v, want /root/users", sp.Users)
	}
	if sp.Data != "/root/.data" {
		t.Errorf("Data = %v, want /root/.data", sp.Data)
	}
	if sp.DB != "/root/.data/foo.db" {
		t.Errorf("DB = %v, want /root/.data/foo.db", sp.DB)
	}
	if sp.ErrorJ != "/root/.data/errors.json" {
		t.Errorf("ErrorJ = %v, want /root/.data/errors.json", sp.ErrorJ)
	}
}

func TestNewStorePath_InvalidPath(t *testing.T) {
	// 测试无效路径
	// 在 Windows 上权限测试行为不同，跳过此测试
	if os.PathSeparator == '\\' {
		t.Skip("Skipping permission test on Windows")
	}

	// 尝试在只读目录中创建
	tempDir, err := os.MkdirTemp("", "path_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 使目录只读
	os.Chmod(tempDir, 0555)
	defer os.Chmod(tempDir, 0755) // 恢复权限以便清理

	readOnlySubDir := filepath.Join(tempDir, "readonly")
	_, err = NewStorePath(readOnlySubDir)
	if err == nil {
		t.Error("NewStorePath() should return error for read-only directory")
	}
}
