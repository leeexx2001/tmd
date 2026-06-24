package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSensitive(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{"空字符串", "", ""},
		{"6字符以下", "abc", "***"},
		{"6字符", "123456", "***"},
		{"7字符", "1234567", "123•••567"},
		{"8字符", "12345678", "123•••678"},
		{"长字符串", "abcdefghij", "abc•••hij"},
		{"中文", "一二三四五六七八", "一二三•••六七八"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskSensitive(tt.input)
			assert.Equal(t, tt.output, got)
		})
	}
}

func TestIsMaskedValue(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		masked bool
	}{
		{"空字符串", "", false},
		{"普通值", "my-real-key", false},
		{"短掩码", "***", true},
		{"长掩码", "abc•••xyz", true},
		{"带 bullet 但不完全是掩码", "•••", true},
		{"值中包含 bullet", "abc•••def", true},
		{"仅 bullet 符号", "•••", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMaskedValue(tt.input)
			assert.Equal(t, tt.masked, got, "isMaskedValue(%q)", tt.input)
		})
	}
}

func TestHandleSaveConfigFields_ClearsAPIKey(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()
	server.config.APIKey = "my-secret-key"

	// __CLEAR__ sentinel 清空 api_key
	body := `{"fields":{"api_key":"__CLEAR__"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/fields", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.handleSaveConfigFields(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "", server.config.APIKey, "APIKey should be cleared")
}

func TestHandleSaveConfigFields_RejectsMaskedValue(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()
	server.config.APIKey = "my-secret-key"

	// 提交掩码值应被拒绝
	body := `{"fields":{"api_key":"abc•••xyz"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/fields", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.handleSaveConfigFields(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code, "masked value should be rejected")

	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "masked placeholder", "error should mention masked placeholder")
}

func TestHandleSaveConfigFields_RejectsShortMaskedValue(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()
	server.config.APIKey = "my-secret-key"

	// 短掩码（***）也应被拒绝
	body := `{"fields":{"api_key":"***"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/fields", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.handleSaveConfigFields(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code, "short masked value should be rejected")
}

func TestHandleSaveConfigFields_APIKeyChangedMessage(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()
	server.config.APIKey = "old-key-1234"

	// 修改 api_key → 应返回立即生效消息
	body := `{"fields":{"api_key":"new-key-5678"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/fields", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.handleSaveConfigFields(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Success bool   `json:"success"`
		Data    struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Data.Message, "API Key has been updated", "should mention API Key immediate effect")
}

func TestHandleSaveConfigFields_OtherFieldChangedMessage(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()
	server.config.APIKey = "my-key-1234"

	// 修改其他字段（如 max_download_routine）→ 应返回需重启消息
	body := `{"fields":{"max_download_routine":"10"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/fields", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.handleSaveConfigFields(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Success bool   `json:"success"`
		Data    struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Data.Message, "restart", "should mention restart requirement")
}

func buildRawConfigYAML(t *testing.T, apiKey string, maxRoutine int) string {
	t.Helper()
	return "root_path: " + t.TempDir() + "\ncookie:\n  auth_token: test\n  ct0: test\napi_key: " + apiKey + "\nmax_download_routine: " + fmt.Sprintf("%d", maxRoutine) + "\nmax_file_name_len: 158"
}

func TestHandleUpdateConfigRaw_APIKeyChangedMessage(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()
	server.config.APIKey = "old-key-1234"

	yamlContent := buildRawConfigYAML(t, "new-key-5678", 5)
	bodyBytes, _ := json.Marshal(map[string]string{"content": yamlContent})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config/raw", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.handleUpdateConfigRaw(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "raw config update should succeed")

	var resp struct {
		Success bool   `json:"success"`
		Data    struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Data.Message, "API Key has been updated", "raw editor should detect API Key change")
}

func TestHandleUpdateConfigRaw_OtherFieldChangedMessage(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()
	server.config.APIKey = "my-key-1234"

	// 修改 max_download_routine 但不改 api_key → 应返回需重启消息
	yamlContent := buildRawConfigYAML(t, "my-key-1234", 10)
	bodyBytes, _ := json.Marshal(map[string]string{"content": yamlContent})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config/raw", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.handleUpdateConfigRaw(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "raw config update should succeed")

	var resp struct {
		Success bool   `json:"success"`
		Data    struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Data.Message, "restart", "should mention restart requirement when API Key unchanged")
}
