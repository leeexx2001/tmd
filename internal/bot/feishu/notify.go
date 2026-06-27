package feishu

import (
	"context"

	log "github.com/sirupsen/logrus"

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
		for openID, taskIDs := range b.userTasks {
			if _, ok := taskIDs[task.ID]; !ok {
				continue
			}
			delete(taskIDs, task.ID)
			if len(taskIDs) == 0 {
				delete(b.userTasks, openID)
			}
			if chatID, ok := b.userChats[openID]; ok {
				b.sendText(chatID, text)
			}
		}
	}
}

func (b *Bot) sendLogAlert(line string) {
	b.mu.Lock()
	chatIDs := make([]string, 0, len(b.userChats))
	for _, chatID := range b.userChats {
		chatIDs = append(chatIDs, chatID)
	}
	b.mu.Unlock()

	for _, chatID := range chatIDs {
		ctx := context.Background()
		_, _, err := b.cli.Message.Send().ToChatID(chatID).SendText(ctx, "🔴 "+line)
		if err != nil {
			log.Warnf("[bot-feishu] Failed to send log notification: %v", err)
		}
	}
}
