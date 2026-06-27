package discord

import (
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
		for channelID, taskIDs := range b.channelTasks {
			if _, ok := taskIDs[task.ID]; !ok {
				continue
			}
			delete(taskIDs, task.ID)
			if len(taskIDs) == 0 {
				delete(b.channelTasks, channelID)
			}
			text := api.FormatTaskResult(task, true)
			_, err := b.session.ChannelMessageSend(channelID, text)
			if err != nil {
				log.Warnf("[bot-discord] Failed to send notification: %v", err)
			}
		}
	}
}

func (b *Bot) sendLogAlert(line string) {
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
