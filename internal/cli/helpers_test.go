package cli

import (
	"testing"

	"github.com/unkmonster/tmd/internal/downloading"
	"github.com/unkmonster/tmd/internal/twitter"
)

func TestTask_Struct(t *testing.T) {
	// 测试 Task 结构体创建
	task := &Task{
		Users: make([]*twitter.User, 0),
		Lists: make([]twitter.ListBase, 0),
	}

	if task.Users == nil {
		t.Error("Task.Users should not be nil")
	}
	if task.Lists == nil {
		t.Error("Task.Lists should not be nil")
	}
}

func TestPrintTask_Empty(t *testing.T) {
	// 测试空任务打印（不 panic 即可）
	task := &Task{
		Users: make([]*twitter.User, 0),
		Lists: make([]twitter.ListBase, 0),
	}

	// 应该正常执行不 panic
	PrintTask(task)
}

func TestPrintTask_WithUsers(t *testing.T) {
	// 测试带用户的任务打印
	task := &Task{
		Users: []*twitter.User{
			{Id: 1, ScreenName: "user1", Name: "User One"},
			{Id: 2, ScreenName: "user2", Name: "User Two"},
		},
		Lists: make([]twitter.ListBase, 0),
	}

	// 应该正常执行不 panic
	PrintTask(task)
}

func TestCLIConfig_TypesFromDownloading(t *testing.T) {
	// 验证 CLIConfig 使用 downloading 包类型
	cfg := CLIConfig{
		UsrArgs:        downloading.UserArgs{},
		ListArgs:       downloading.ListArgs{},
		FollArgs:       downloading.UserArgs{},
		ProfileUsers:   downloading.UserArgs{},
		ProfileList:    downloading.ListArgs{},
		JsonArgs:       downloading.JsonPathsArgs{},
		AutoFollow:     false,
		NoRetry:        false,
		MarkDownloaded: false,
		MarkTime:       "",
		NoProfile:      false,
	}

	// 测试 UserArgs 类型
	cfg.UsrArgs.ID = []uint64{1, 2, 3}
	cfg.UsrArgs.ScreenName = []string{"user1", "user2"}

	if len(cfg.UsrArgs.ID) != 3 {
		t.Errorf("UsrArgs.ID length = %d, want 3", len(cfg.UsrArgs.ID))
	}
	if len(cfg.UsrArgs.ScreenName) != 2 {
		t.Errorf("UsrArgs.ScreenName length = %d, want 2", len(cfg.UsrArgs.ScreenName))
	}

	// 测试 ListArgs 类型
	cfg.ListArgs.ID = []uint64{100, 200}
	if len(cfg.ListArgs.ID) != 2 {
		t.Errorf("ListArgs.ID length = %d, want 2", len(cfg.ListArgs.ID))
	}

	// 测试 JsonPathsArgs 类型
	cfg.JsonArgs.Paths = []string{"file1.json", "file2.json"}
	if len(cfg.JsonArgs.Paths) != 2 {
		t.Errorf("JsonPathsArgs.Paths length = %d, want 2", len(cfg.JsonArgs.Paths))
	}
}

func TestUserArgs_Set(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantID       []uint64
		wantScreen   []string
	}{
		{
			name:       "screen name",
			input:      "elonmusk",
			wantID:     nil,
			wantScreen: []string{"elonmusk"},
		},
		{
			name:       "screen name with @",
			input:      "@elonmusk",
			wantID:     nil,
			wantScreen: []string{"elonmusk"},
		},
		{
			name:       "numeric ID",
			input:      "44196397",
			wantID:     []uint64{44196397},
			wantScreen: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ua := &downloading.UserArgs{}
			err := ua.Set(tt.input)
			if err != nil {
				t.Errorf("Set() error = %v", err)
				return
			}

			if tt.wantID != nil {
				if len(ua.ID) != len(tt.wantID) || ua.ID[0] != tt.wantID[0] {
					t.Errorf("ID = %v, want %v", ua.ID, tt.wantID)
				}
			}
			if tt.wantScreen != nil {
				if len(ua.ScreenName) != len(tt.wantScreen) || ua.ScreenName[0] != tt.wantScreen[0] {
					t.Errorf("ScreenName = %v, want %v", ua.ScreenName, tt.wantScreen)
				}
			}
		})
	}
}

func TestUserArgs_Set_Multiple(t *testing.T) {
	ua := &downloading.UserArgs{}

	// 添加多个值
	_ = ua.Set("elonmusk")
	_ = ua.Set("44196397")
	_ = ua.Set("@NASA")

	if len(ua.ScreenName) != 2 {
		t.Errorf("ScreenName length = %d, want 2", len(ua.ScreenName))
	}
	if len(ua.ID) != 1 {
		t.Errorf("ID length = %d, want 1", len(ua.ID))
	}

	// 验证值
	if ua.ScreenName[0] != "elonmusk" || ua.ScreenName[1] != "NASA" {
		t.Errorf("ScreenName = %v, want [elonmusk NASA]", ua.ScreenName)
	}
	if ua.ID[0] != 44196397 {
		t.Errorf("ID[0] = %d, want 44196397", ua.ID[0])
	}
}

func TestUserArgs_String(t *testing.T) {
	ua := &downloading.UserArgs{
		ID:         []uint64{1, 2},
		ScreenName: []string{"user1", "user2"},
	}

	s := ua.String()
	if s == "" {
		t.Error("String() returned empty string")
	}
}

func TestListArgs_Set(t *testing.T) {
	la := &downloading.ListArgs{}

	err := la.Set("12345")
	if err != nil {
		t.Errorf("Set() error = %v", err)
		return
	}

	if len(la.ID) != 1 || la.ID[0] != 12345 {
		t.Errorf("ID = %v, want [12345]", la.ID)
	}
}

func TestListArgs_Set_Invalid(t *testing.T) {
	la := &downloading.ListArgs{}

	err := la.Set("not-a-number")
	if err == nil {
		t.Error("Set() should return error for invalid input")
	}
}

func TestListArgs_Set_Multiple(t *testing.T) {
	la := &downloading.ListArgs{}

	_ = la.Set("111")
	_ = la.Set("222")
	_ = la.Set("333")

	if len(la.ID) != 3 {
		t.Errorf("ID length = %d, want 3", len(la.ID))
	}
}

func TestListArgs_String(t *testing.T) {
	la := &downloading.ListArgs{
		ID: []uint64{1, 2, 3},
	}

	s := la.String()
	if s == "" {
		t.Error("String() returned empty string")
	}
}

func TestJsonPathsArgs_Set(t *testing.T) {
	ja := &downloading.JsonPathsArgs{}

	err := ja.Set("/path/to/file.json")
	if err != nil {
		t.Errorf("Set() error = %v", err)
		return
	}

	if len(ja.Paths) != 1 || ja.Paths[0] != "/path/to/file.json" {
		t.Errorf("Paths = %v, want [/path/to/file.json]", ja.Paths)
	}
}

func TestJsonPathsArgs_Set_Multiple(t *testing.T) {
	ja := &downloading.JsonPathsArgs{}

	_ = ja.Set("file1.json")
	_ = ja.Set("file2.json")
	_ = ja.Set("file3.json")

	if len(ja.Paths) != 3 {
		t.Errorf("Paths length = %d, want 3", len(ja.Paths))
	}
}

func TestJsonPathsArgs_String(t *testing.T) {
	ja := &downloading.JsonPathsArgs{
		Paths: []string{"file1.json", "file2.json"},
	}

	s := ja.String()
	if s == "" {
		t.Error("String() returned empty string")
	}
}

func TestJsonPathsArgs_GetPaths(t *testing.T) {
	ja := &downloading.JsonPathsArgs{
		Paths: []string{"file1.json", "file2.json"},
	}

	paths := ja.GetPaths()
	if len(paths) != 2 {
		t.Errorf("GetPaths() length = %d, want 2", len(paths))
	}
}
