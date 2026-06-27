package feishu

import (
	"context"
	"strings"

	"github.com/chyroc/lark"
	log "github.com/sirupsen/logrus"
)

func (b *Bot) handleMessage(ctx context.Context, event *lark.EventV2IMMessageReceiveV1) {
	if event.Message.MessageType != lark.MsgTypeText {
		return
	}
	content, err := lark.UnwrapMessageContent(event.Message.MessageType, event.Message.Content)
	if err != nil {
		log.Warnf("[bot-feishu] Failed to unwrap message: %v", err)
		return
	}
	text := strings.TrimSpace(content.Text.Text)
	if text == "" || !strings.HasPrefix(text, "/") {
		return
	}

	// 权限检查
	openID := event.Sender.SenderID.OpenID
	if !b.isAllowed(openID) {
		b.sendReply(event.Message.MessageID, "⛔ Unauthorized")
		return
	}

	// 保存 chat_id 用于主动推送
	b.mu.Lock()
	b.userChats[openID] = event.Message.ChatID
	b.mu.Unlock()

	// 解析命令
	parts := strings.SplitN(text, " ", 2)
	cmd := strings.TrimPrefix(parts[0], "/")
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	switch cmd {
	case "dl", "download":
		b.cmdDownload(event.Message.MessageID, args, openID)
	case "status":
		b.cmdStatus(event.Message.MessageID, args)
	case "cancel":
		b.cmdCancel(event.Message.MessageID, args)
	case "tasks":
		b.cmdTasks(event.Message.MessageID)
	case "start", "help":
		b.cmdHelp(event.Message.MessageID)
		
	default:
		b.sendReply(event.Message.MessageID, "Unknown command. Send /help for available commands.")
	}
}
