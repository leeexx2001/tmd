package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/unkmonster/tmd/internal/api"
	"github.com/unkmonster/tmd/internal/config"
)

func TestBot_IsAllowed(t *testing.T) {
	cfg := &config.TelegramBotConfig{AllowedUsers: nil}
	bot := &Bot{config: cfg}
	assert.True(t, bot.isAllowed(12345))

	cfg.AllowedUsers = []int64{100, 200}
	bot.config = cfg
	assert.True(t, bot.isAllowed(100))
	assert.False(t, bot.isAllowed(300))
}

func TestBot_FormatTaskResult(t *testing.T) {
	t.Run("completed", func(t *testing.T) {
		task := &api.Task{
			ID: "task_test", Status: api.TaskStatusCompleted,
			Result: &api.TaskResult{Main: &api.TaskMainResult{Downloaded: 10, Failed: 1}},
		}
		result := api.FormatTaskResult(task, true)
		assert.Contains(t, result, "✅")
		assert.Contains(t, result, "Downloaded: 10")
	})

	t.Run("failed", func(t *testing.T) {
		task := &api.Task{
			ID: "task_fail", Status: api.TaskStatusFailed,
			Error: "something went wrong",
		}
		result := api.FormatTaskResult(task, true)
		assert.Contains(t, result, "❌")
		assert.Contains(t, result, "something went wrong")
	})

	t.Run("no result", func(t *testing.T) {
		task := &api.Task{
			ID: "task_none", Status: api.TaskStatusCompleted,
		}
		result := api.FormatTaskResult(task, true)
		assert.Contains(t, result, "✅")
	})
}

func TestBot_EscapeMD(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"err: something_bad", `err: something\_bad`},
		{"error: *panic*", `error: \*panic\*`},
		{"backtick`here", "backtick'here"},
		{"[warn]", `\[warn]`},
	}
	for _, tt := range tests {
		got := escapeMD(tt.input)
		assert.Equal(t, tt.expected, got, "escapeMD(%q)", tt.input)
	}
}

func TestBot_StatusConstants(t *testing.T) {
	assert.Equal(t, api.TaskStatus("completed"), api.TaskStatusCompleted)
	assert.Equal(t, api.TaskStatus("failed"), api.TaskStatusFailed)
	assert.Equal(t, api.TaskStatus("cancelled"), api.TaskStatusCancelled)
}
