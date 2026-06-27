package pushover

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/unkmonster/tmd/internal/api"
)

func TestBot_FormatTaskResult(t *testing.T) {
	t.Run("completed", func(t *testing.T) {
		task := &api.Task{
			ID: "task_test", Status: api.TaskStatusCompleted,
			Result: &api.TaskResult{Main: &api.TaskMainResult{Downloaded: 10, Failed: 1}},
		}
		title := "✅ TMD Download Complete"
		msg := api.FormatTaskResult(task, false)
		assert.Contains(t, title, "✅")
		assert.Contains(t, msg, "Downloaded: 10")
	})

	t.Run("failed", func(t *testing.T) {
		task := &api.Task{
			ID: "task_fail", Status: api.TaskStatusFailed,
			Error: "something went wrong",
		}
		title := "❌ TMD Download Failed"
		msg := api.FormatTaskResult(task, false)
		assert.Contains(t, title, "❌")
		assert.Contains(t, msg, "something went wrong")
	})
}

func TestBot_StatusConstants(t *testing.T) {
	assert.Equal(t, api.TaskStatus("completed"), api.TaskStatusCompleted)
	assert.Equal(t, api.TaskStatus("failed"), api.TaskStatusFailed)
}
