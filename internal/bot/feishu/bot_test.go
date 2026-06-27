package feishu

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/unkmonster/tmd/internal/api"
	"github.com/unkmonster/tmd/internal/config"
)

func TestBot_IsAllowed(t *testing.T) {
	cfg := &config.FeishuBotConfig{AllowedUsers: nil}
	bot := &Bot{config: cfg}
	assert.True(t, bot.isAllowed("ou_xxx"))

	cfg.AllowedUsers = []string{"ou_alice", "ou_bob"}
	bot.config = cfg
	assert.True(t, bot.isAllowed("ou_alice"))
	assert.False(t, bot.isAllowed("ou_charlie"))
}

func TestBot_FormatTaskResult(t *testing.T) {
	t.Run("completed", func(t *testing.T) {
		task := &api.Task{
			ID: "task_test", Status: api.TaskStatusCompleted,
			Result: &api.TaskResult{Main: &api.TaskMainResult{Downloaded: 10, Failed: 1}},
		}
		result := api.FormatTaskResult(task, false)
		assert.Contains(t, result, "✅")
		assert.Contains(t, result, "Downloaded: 10")
	})

	t.Run("failed", func(t *testing.T) {
		task := &api.Task{
			ID: "task_fail", Status: api.TaskStatusFailed,
			Error: "something went wrong",
		}
		result := api.FormatTaskResult(task, false)
		assert.Contains(t, result, "❌")
		assert.Contains(t, result, "something went wrong")
	})
}

func TestBot_ParseDLArgs(t *testing.T) {
	t.Run("bare user", func(t *testing.T) {
		dlType, target := parseDLArgs("elonmusk")
		assert.Equal(t, "user", dlType)
		assert.Equal(t, "elonmusk", target)
	})
	t.Run("user type", func(t *testing.T) {
		dlType, target := parseDLArgs("user elonmusk")
		assert.Equal(t, "user", dlType)
		assert.Equal(t, "elonmusk", target)
	})
	t.Run("list type", func(t *testing.T) {
		dlType, target := parseDLArgs("list 12345")
		assert.Equal(t, "list", dlType)
		assert.Equal(t, "12345", target)
	})
	t.Run("foll type", func(t *testing.T) {
		dlType, target := parseDLArgs("foll elonmusk")
		assert.Equal(t, "foll", dlType)
		assert.Equal(t, "elonmusk", target)
	})
}

func TestBot_StatusConstants(t *testing.T) {
	assert.Equal(t, api.TaskStatus("completed"), api.TaskStatusCompleted)
	assert.Equal(t, api.TaskStatus("failed"), api.TaskStatusFailed)
}
