package discord

import (
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
		for channelID, taskIDs := range b.channelTasks {
			if _, ok := taskIDs[task.ID]; !ok {
				continue
			}
			delete(taskIDs, task.ID)
			if len(taskIDs) == 0 {
				delete(b.channelTasks, channelID)
			}
			text := b.formatTaskResult(task)
			_, err := b.session.ChannelMessageSend(channelID, text)
			if err != nil {
				log.Warnf("[bot-discord] Failed to send notification: %v", err)
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
			for _, userID := range b.config.AllowedUsers {
				channel, err := b.session.UserChannelCreate(userID)
				if err != nil {
					log.Warnf("[bot-discord] Failed to create DM channel: %v", err)
					continue
				}
				_, err = b.session.ChannelMessageSend(channel.ID, "🔴 `"+escapeDiscord(line)+"`")
				if err != nil {
					log.Warnf("[bot-discord] Failed to send log notification: %v", err)
				}
			}
		}
	}
}

func escapeDiscord(s string) string {
	result := make([]byte, 0, len(s)*2)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '_', '*', '`', '~', '|', '>':
			result = append(result, '\\', c)
		default:
			result = append(result, c)
		}
	}
	return string(result)
}
