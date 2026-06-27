package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleUpdates() {
	defer b.wg.Done()
	for {
		select {
		case <-b.stopCh:
			return
		case update, ok := <-b.updates:
			if !ok {
				return
			}
			if update.Message == nil || !update.Message.IsCommand() {
				continue
			}
			if !b.isAllowed(update.Message.From.ID) {
				b.sendText(update.Message.Chat.ID, "⛔ Unauthorized")
				continue
			}
			b.handleCommand(update.Message)
		}
	}
}

func (b *Bot) isAllowed(userID int64) bool {
	if len(b.config.AllowedUsers) == 0 {
		return true
	}
	for _, id := range b.config.AllowedUsers {
		if id == userID {
			return true
		}
	}
	return false
}

func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		b.cmdStart(msg)
	case "dl", "download":
		b.cmdDownload(msg)
	case "status":
		b.cmdStatus(msg)
	case "cancel":
		b.cmdCancel(msg)
	case "tasks":
		b.cmdTasks(msg)
	case "help":
		b.cmdHelp(msg)
	default:
		b.sendText(msg.Chat.ID, "Unknown command. Send /help for available commands.")
	}
}
