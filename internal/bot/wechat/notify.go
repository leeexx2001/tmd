package wechat

import (
	"context"

	"github.com/unkmonster/tmd/internal/api"
)

func (b *Bot) notifyTaskChanges(data interface{}) {
	tasks, ok := data.([]*api.Task)
	if !ok {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, task := range tasks {
		if task.Status != api.TaskStatusCompleted && task.Status != api.TaskStatusFailed {
			continue
		}
		text := api.FormatTaskResult(task, false)
		for userID, taskIDs := range b.userTasks {
			if _, ok := taskIDs[task.ID]; !ok {
				continue
			}
			delete(taskIDs, task.ID)
			if len(taskIDs) == 0 {
				delete(b.userTasks, userID)
			}
			ctx := context.Background()
			b.sendText(ctx, userID, text)
		}
	}
}

func (b *Bot) sendLogAlert(line string) {
	b.mu.Lock()
	userIDs := make([]string, 0, len(b.userTokens))
	for uid := range b.userTokens {
		userIDs = append(userIDs, uid)
	}
	b.mu.Unlock()

	for _, userID := range userIDs {
		ctx := context.Background()
		b.sendText(ctx, userID, "🔴 "+line)
	}
}
