package wechat

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/unkmonster/tmd/internal/api"
	"github.com/unkmonster/tmd/internal/config"
)

func TestBot_IsAllowed(t *testing.T) {
	cfg := &config.WeChatBotConfig{AllowedUsers: nil}
	bot := &Bot{config: cfg}
	assert.True(t, bot.isAllowed("user@im.wechat"))

	cfg.AllowedUsers = []string{"alice@im.wechat", "bob@im.wechat"}
	bot.config = cfg
	assert.True(t, bot.isAllowed("alice@im.wechat"))
	assert.False(t, bot.isAllowed("charlie@im.wechat"))
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
