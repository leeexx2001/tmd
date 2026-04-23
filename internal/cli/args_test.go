package cli

import (
	"testing"

	"github.com/unkmonster/tmd/internal/downloading"
)

func TestParseArgs_Empty(t *testing.T) {
	fs, cfg, err := ParseArgs([]string{})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}
	if fs == nil {
		t.Error("ParseArgs() fs is nil")
		return
	}
	if cfg == nil {
		t.Error("ParseArgs() cfg is nil")
		return
	}
}

func TestParseArgs_User(t *testing.T) {
	_, cfg, err := ParseArgs([]string{"-user", "elonmusk"})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	if len(cfg.UsrArgs.ScreenName) != 1 || cfg.UsrArgs.ScreenName[0] != "elonmusk" {
		t.Errorf("UsrArgs.ScreenName = %v, want [elonmusk]", cfg.UsrArgs.ScreenName)
	}
}

func TestParseArgs_UserWithAt(t *testing.T) {
	_, cfg, err := ParseArgs([]string{"-user", "@elonmusk"})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	if len(cfg.UsrArgs.ScreenName) != 1 || cfg.UsrArgs.ScreenName[0] != "elonmusk" {
		t.Errorf("UsrArgs.ScreenName = %v, want [elonmusk]", cfg.UsrArgs.ScreenName)
	}
}

func TestParseArgs_UserID(t *testing.T) {
	_, cfg, err := ParseArgs([]string{"-user", "44196397"})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	if len(cfg.UsrArgs.ID) != 1 || cfg.UsrArgs.ID[0] != 44196397 {
		t.Errorf("UsrArgs.ID = %v, want [44196397]", cfg.UsrArgs.ID)
	}
}

func TestParseArgs_MultipleUsers(t *testing.T) {
	_, cfg, err := ParseArgs([]string{"-user", "elonmusk", "-user", "NASA"})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	if len(cfg.UsrArgs.ScreenName) != 2 {
		t.Errorf("UsrArgs.ScreenName length = %d, want 2", len(cfg.UsrArgs.ScreenName))
	}
}

func TestParseArgs_List(t *testing.T) {
	_, cfg, err := ParseArgs([]string{"-list", "123456789"})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	if len(cfg.ListArgs.ID) != 1 || cfg.ListArgs.ID[0] != 123456789 {
		t.Errorf("ListArgs.ID = %v, want [123456789]", cfg.ListArgs.ID)
	}
}

func TestParseArgs_MultipleLists(t *testing.T) {
	_, cfg, err := ParseArgs([]string{"-list", "111", "-list", "222"})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	if len(cfg.ListArgs.ID) != 2 {
		t.Errorf("ListArgs.ID length = %d, want 2", len(cfg.ListArgs.ID))
	}
}

func TestParseArgs_Foll(t *testing.T) {
	_, cfg, err := ParseArgs([]string{"-foll", "testuser"})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	if len(cfg.FollArgs.ScreenName) != 1 || cfg.FollArgs.ScreenName[0] != "testuser" {
		t.Errorf("FollArgs.ScreenName = %v, want [testuser]", cfg.FollArgs.ScreenName)
	}
}

func TestParseArgs_Json(t *testing.T) {
	_, cfg, err := ParseArgs([]string{"-json", "/path/to/file.json"})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	if len(cfg.JsonArgs.Paths) != 1 || cfg.JsonArgs.Paths[0] != "/path/to/file.json" {
		t.Errorf("JsonArgs.Paths = %v, want [/path/to/file.json]", cfg.JsonArgs.Paths)
	}
}

func TestParseArgs_MultipleJson(t *testing.T) {
	_, cfg, err := ParseArgs([]string{"-json", "file1.json", "-json", "file2.json"})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	if len(cfg.JsonArgs.Paths) != 2 {
		t.Errorf("JsonArgs.Paths length = %d, want 2", len(cfg.JsonArgs.Paths))
	}
}

func TestParseArgs_ProfileUser(t *testing.T) {
	_, cfg, err := ParseArgs([]string{"-profile-user", "elonmusk"})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	if len(cfg.ProfileUsers.ScreenName) != 1 || cfg.ProfileUsers.ScreenName[0] != "elonmusk" {
		t.Errorf("ProfileUsers.ScreenName = %v, want [elonmusk]", cfg.ProfileUsers.ScreenName)
	}
}

func TestParseArgs_ProfileList(t *testing.T) {
	_, cfg, err := ParseArgs([]string{"-profile-list", "12345"})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	if len(cfg.ProfileList.ID) != 1 || cfg.ProfileList.ID[0] != 12345 {
		t.Errorf("ProfileList.ID = %v, want [12345]", cfg.ProfileList.ID)
	}
}

func TestParseArgs_BoolFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		checkFn  func(*CLIConfig) bool
		wantDesc string
	}{
		{
			name: "auto-follow",
			args: []string{"-auto-follow"},
			checkFn: func(cfg *CLIConfig) bool {
				return cfg.AutoFollow
			},
			wantDesc: "AutoFollow = true",
		},
		{
			name: "no-retry",
			args: []string{"-no-retry"},
			checkFn: func(cfg *CLIConfig) bool {
				return cfg.NoRetry
			},
			wantDesc: "NoRetry = true",
		},
		{
			name: "mark-downloaded",
			args: []string{"-mark-downloaded"},
			checkFn: func(cfg *CLIConfig) bool {
				return cfg.MarkDownloaded
			},
			wantDesc: "MarkDownloaded = true",
		},
		{
			name: "noprofile",
			args: []string{"-noprofile"},
			checkFn: func(cfg *CLIConfig) bool {
				return cfg.NoProfile
			},
			wantDesc: "NoProfile = true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cfg, err := ParseArgs(tt.args)
			if err != nil {
				t.Errorf("ParseArgs() error = %v", err)
				return
			}
			if !tt.checkFn(cfg) {
				t.Errorf("%s", tt.wantDesc)
			}
		})
	}
}

func TestParseArgs_MarkTime(t *testing.T) {
	_, cfg, err := ParseArgs([]string{"-mark-time", "2024-01-01T00:00:00"})
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	if cfg.MarkTime != "2024-01-01T00:00:00" {
		t.Errorf("MarkTime = %v, want 2024-01-01T00:00:00", cfg.MarkTime)
	}
}

func TestParseArgs_Complex(t *testing.T) {
	args := []string{
		"-user", "elonmusk",
		"-list", "123456",
		"-foll", "NASA",
		"-json", "tweets.json",
		"-auto-follow",
		"-no-retry",
		"-noprofile",
	}

	_, cfg, err := ParseArgs(args)
	if err != nil {
		t.Errorf("ParseArgs() error = %v", err)
		return
	}

	// 验证所有参数都被正确解析
	if len(cfg.UsrArgs.ScreenName) != 1 || cfg.UsrArgs.ScreenName[0] != "elonmusk" {
		t.Error("UsrArgs not parsed correctly")
	}
	if len(cfg.ListArgs.ID) != 1 || cfg.ListArgs.ID[0] != 123456 {
		t.Error("ListArgs not parsed correctly")
	}
	if len(cfg.FollArgs.ScreenName) != 1 || cfg.FollArgs.ScreenName[0] != "NASA" {
		t.Error("FollArgs not parsed correctly")
	}
	if len(cfg.JsonArgs.Paths) != 1 || cfg.JsonArgs.Paths[0] != "tweets.json" {
		t.Error("JsonArgs not parsed correctly")
	}
	if !cfg.AutoFollow {
		t.Error("AutoFollow not parsed correctly")
	}
	if !cfg.NoRetry {
		t.Error("NoRetry not parsed correctly")
	}
	if !cfg.NoProfile {
		t.Error("NoProfile not parsed correctly")
	}
}

// 测试 CLIConfig 类型使用 downloading 包类型
func TestCLIConfig_Types(t *testing.T) {
	cfg := &CLIConfig{
		UsrArgs:      downloading.UserArgs{},
		ListArgs:     downloading.ListArgs{},
		FollArgs:     downloading.UserArgs{},
		ProfileUsers: downloading.UserArgs{},
		ProfileList:  downloading.ListArgs{},
		JsonArgs:     downloading.JsonPathsArgs{},
	}

	// 验证类型正确
	_ = cfg.UsrArgs.Set("testuser")
	_ = cfg.ListArgs.Set("12345")
	_ = cfg.JsonArgs.Set("test.json")

	if len(cfg.UsrArgs.ScreenName) != 1 {
		t.Error("UserArgs.Set not working")
	}
	if len(cfg.ListArgs.ID) != 1 {
		t.Error("ListArgs.Set not working")
	}
	if len(cfg.JsonArgs.Paths) != 1 {
		t.Error("JsonPathsArgs.Set not working")
	}
}
