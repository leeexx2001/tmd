package feishu

import (
	"context"
	"fmt"
	"strings"

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
		text := b.formatTaskResult(task)
		for _, chatID := range b.userChats {
			b.sendText(chatID, text)
		}
	}
}

func (b *Bot) formatTaskResult(task *api.Task) string {
	icon := "✅"
	if task.Status == api.TaskStatusFailed {
		icon = "❌"
	}
	text := fmt.Sprintf("%s Task %s %s", icon, task.ID, task.Status)
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
			b.mu.Lock()
			for _, chatID := range b.userChats {
				ctx := context.Background()
				_, _, err := b.cli.Message.Send().ToChatID(chatID).SendText(ctx, "🔴 "+line)
				if err != nil {
					log.Warnf("[bot-feishu] Failed to send log notification: %v", err)
				}
			}
			b.mu.Unlock()
		}
	}
}
