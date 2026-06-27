package telegram

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/api"
)

func (b *Bot) handleEvents() {
	defer b.wg.Done()
	ch, unsub := b.eventBus.Subscribe()
	defer unsub()
	for {
		select {
		case <-b.stopCh:
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			if evt.Event != "tasks" {
				continue
			}
			b.notifyTaskChanges(evt.Data)
		}
	}
}

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
		for chatID, taskIDs := range b.chatTasks {
			if _, ok := taskIDs[task.ID]; !ok {
				continue
			}
			delete(taskIDs, task.ID)
			if len(taskIDs) == 0 {
				delete(b.chatTasks, chatID)
			}
			text := b.formatTaskResult(task)
			msg := tgbotapi.NewMessage(chatID, text)
			msg.ParseMode = "markdown"
			if _, err := b.api.Send(msg); err != nil {
				log.Warnf("[bot-telegram] Failed to send notification: %v", err)
			}
		}
	}
}

func (b *Bot) formatTaskResult(task *api.Task) string {
	icon := "✅"
	if task.Status == api.TaskStatusFailed {
		icon = "❌"
	}
	text := fmt.Sprintf("%s Task `%s` %s", icon, task.ID, task.Status)
	if task.Result != nil && task.Result.Main != nil {
		text += fmt.Sprintf("\nDownloaded: %d, Failed: %d", task.Result.Main.Downloaded, task.Result.Main.Failed)
	}
	if task.Error != "" {
		text += fmt.Sprintf("\nError: %s", task.Error)
	}
	return text
}

func (b *Bot) handleLogs() {
	defer b.wg.Done()
	ch, unsub := b.logHub.Subscribe()
	defer unsub()
	for {
		select {
		case <-b.stopCh:
			return
		case line, ok := <-ch:
			if !ok {
				return
			}
			if !strings.Contains(line, "level=error") && !strings.Contains(line, "level=fatal") {
				continue
			}
			for _, chatID := range b.config.AllowedUsers {
				msg := tgbotapi.NewMessage(chatID, "🔴 `"+escapeMD(line)+"`")
				msg.ParseMode = "markdown"
				if _, err := b.api.Send(msg); err != nil {
					log.Warnf("[bot-telegram] Failed to send log notification: %v", err)
				}
			}
		}
	}
}

func escapeMD(s string) string {
	s = strings.ReplaceAll(s, "_", "\\_")
	s = strings.ReplaceAll(s, "*", "\\*")
	s = strings.ReplaceAll(s, "`", "'")
	s = strings.ReplaceAll(s, "[", "\\[")
	return s
}
