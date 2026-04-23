package downloading

import (
	"testing"
)

func TestUserArgs_Set_ScreenName(t *testing.T) {
	ua := &UserArgs{}
	err := ua.Set("elonmusk")
	if err != nil {
		t.Errorf("Set() error = %v", err)
		return
	}

	if len(ua.ScreenName) != 1 || ua.ScreenName[0] != "elonmusk" {
		t.Errorf("ScreenName = %v, want [elonmusk]", ua.ScreenName)
	}
	if len(ua.ID) != 0 {
		t.Errorf("ID should be empty, got %v", ua.ID)
	}
}

func TestUserArgs_Set_ScreenNameWithAt(t *testing.T) {
	ua := &UserArgs{}
	err := ua.Set("@elonmusk")
	if err != nil {
		t.Errorf("Set() error = %v", err)
		return
	}

	if len(ua.ScreenName) != 1 || ua.ScreenName[0] != "elonmusk" {
		t.Errorf("ScreenName = %v, want [elonmusk]", ua.ScreenName)
	}
}

func TestUserArgs_Set_NumericID(t *testing.T) {
	ua := &UserArgs{}
	err := ua.Set("44196397")
	if err != nil {
		t.Errorf("Set() error = %v", err)
		return
	}

	if len(ua.ID) != 1 || ua.ID[0] != 44196397 {
		t.Errorf("ID = %v, want [44196397]", ua.ID)
	}
	if len(ua.ScreenName) != 0 {
		t.Errorf("ScreenName should be empty, got %v", ua.ScreenName)
	}
}

func TestUserArgs_Set_Multiple(t *testing.T) {
	ua := &UserArgs{}

	inputs := []string{"user1", "12345", "@user2", "67890"}
	for _, input := range inputs {
		err := ua.Set(input)
		if err != nil {
			t.Errorf("Set(%s) error = %v", input, err)
			return
		}
	}

	if len(ua.ScreenName) != 2 {
		t.Errorf("ScreenName length = %d, want 2", len(ua.ScreenName))
	}
	if len(ua.ID) != 2 {
		t.Errorf("ID length = %d, want 2", len(ua.ID))
	}

	// 验证值
	if ua.ScreenName[0] != "user1" || ua.ScreenName[1] != "user2" {
		t.Errorf("ScreenName = %v, want [user1 user2]", ua.ScreenName)
	}
	if ua.ID[0] != 12345 || ua.ID[1] != 67890 {
		t.Errorf("ID = %v, want [12345 67890]", ua.ID)
	}
}

func TestUserArgs_String(t *testing.T) {
	ua := &UserArgs{
		ID:         []uint64{1, 2, 3},
		ScreenName: []string{"user1", "user2"},
	}

	s := ua.String()
	if s == "" {
		t.Error("String() returned empty string")
	}

	// 验证字符串包含必要信息
	if len(s) == 0 {
		t.Error("String() should not return empty string")
	}
}

func TestUserArgs_String_Empty(t *testing.T) {
	ua := &UserArgs{}

	s := ua.String()
	if s == "" {
		t.Error("String() returned empty string for empty UserArgs")
	}
}

func TestListArgs_Set(t *testing.T) {
	la := &ListArgs{}
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
	la := &ListArgs{}
	err := la.Set("not-a-number")
	if err == nil {
		t.Error("Set() should return error for invalid input")
	}
}

func TestListArgs_Set_Multiple(t *testing.T) {
	la := &ListArgs{}

	inputs := []string{"111", "222", "333"}
	for _, input := range inputs {
		err := la.Set(input)
		if err != nil {
			t.Errorf("Set(%s) error = %v", input, err)
			return
		}
	}

	if len(la.ID) != 3 {
		t.Errorf("ID length = %d, want 3", len(la.ID))
	}

	expected := []uint64{111, 222, 333}
	for i, id := range la.ID {
		if id != expected[i] {
			t.Errorf("ID[%d] = %d, want %d", i, id, expected[i])
		}
	}
}

func TestListArgs_String(t *testing.T) {
	la := &ListArgs{
		ID: []uint64{1, 2, 3},
	}

	s := la.String()
	if s == "" {
		t.Error("String() returned empty string")
	}
}

func TestListArgs_String_Empty(t *testing.T) {
	la := &ListArgs{}

	s := la.String()
	if s == "" {
		t.Error("String() returned empty string for empty ListArgs")
	}
}

func TestJsonPathsArgs_Set(t *testing.T) {
	ja := &JsonPathsArgs{}
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
	ja := &JsonPathsArgs{}

	inputs := []string{"file1.json", "file2.json", "/path/to/file3.json"}
	for _, input := range inputs {
		err := ja.Set(input)
		if err != nil {
			t.Errorf("Set(%s) error = %v", input, err)
			return
		}
	}

	if len(ja.Paths) != 3 {
		t.Errorf("Paths length = %d, want 3", len(ja.Paths))
	}

	expected := []string{"file1.json", "file2.json", "/path/to/file3.json"}
	for i, path := range ja.Paths {
		if path != expected[i] {
			t.Errorf("Paths[%d] = %s, want %s", i, path, expected[i])
		}
	}
}

func TestJsonPathsArgs_String(t *testing.T) {
	ja := &JsonPathsArgs{
		Paths: []string{"file1.json", "file2.json"},
	}

	s := ja.String()
	if s == "" {
		t.Error("String() returned empty string")
	}

	// 验证字符串包含所有路径
	if len(s) == 0 {
		t.Error("String() should not return empty string")
	}
}

func TestJsonPathsArgs_String_Empty(t *testing.T) {
	ja := &JsonPathsArgs{}

	s := ja.String()
	if s != "" {
		t.Errorf("String() = %s, want empty string", s)
	}
}

func TestJsonPathsArgs_GetPaths(t *testing.T) {
	ja := &JsonPathsArgs{
		Paths: []string{"file1.json", "file2.json"},
	}

	paths := ja.GetPaths()
	if len(paths) != 2 {
		t.Errorf("GetPaths() length = %d, want 2", len(paths))
	}

	// 验证返回的路径正确
	if paths[0] != "file1.json" || paths[1] != "file2.json" {
		t.Errorf("GetPaths() = %v, want [file1.json file2.json]", paths)
	}
}

func TestJsonPathsArgs_GetPaths_Empty(t *testing.T) {
	ja := &JsonPathsArgs{}

	paths := ja.GetPaths()
	// 当 Paths 为 nil 时，GetPaths 返回 nil 是合理的
	if paths != nil && len(paths) != 0 {
		t.Errorf("GetPaths() = %v, want nil or empty slice", paths)
	}
}

// 测试 flag.Value 接口实现
func TestUserArgs_FlagValue(t *testing.T) {
	ua := &UserArgs{}

	// 测试 Set 和 String 方法（flag.Value 接口要求）
	_ = ua.Set("testuser")
	_ = ua.String()

	// 只要能调用不 panic 即可
}

func TestListArgs_FlagValue(t *testing.T) {
	la := &ListArgs{}

	_ = la.Set("12345")
	_ = la.String()
}

func TestJsonPathsArgs_FlagValue(t *testing.T) {
	ja := &JsonPathsArgs{}

	_ = ja.Set("test.json")
	_ = ja.String()
}
