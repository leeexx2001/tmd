package telegram

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/unkmonster/tmd/internal/api"
)

func (b *Bot) cmdStart(msg *tgbotapi.Message) {
	b.sendText(msg.Chat.ID, "TMD Bot running.\n/dl [user|list|foll] <target> — download\n/status <id> — task status\n/cancel <id> — cancel task\n/tasks — list tasks\n/help — this message")
}

func (b *Bot) cmdDownload(msg *tgbotapi.Message) {
	raw := strings.TrimSpace(msg.CommandArguments())
	if raw == "" {
		b.sendText(msg.Chat.ID, "Usage: /dl [user|list|foll] <target>\nDefaults to user if type omitted.\nExamples:\n/dl elonmusk\n/dl list 12345\n/dl foll elonmusk")
		return
	}
	dlType, target := parseDLArgs(raw)
	if target == "" {
		b.sendText(msg.Chat.ID, "Usage: /dl [user|list|foll] <target>")
		return
	}
	var task *api.Task
	switch dlType {
	case "list":
		listID, err := strconv.ParseUint(target, 10, 64)
		if err != nil {
			b.sendText(msg.Chat.ID, "Invalid list ID. Must be a number.")
			return
		}
		task = b.taskManager.CreateTask(api.TaskTypeListDownload, &api.ListDownloadTaskData{
			ListID: api.StringUint64(listID),
		})
	case "foll":
		task = b.taskManager.CreateTask(api.TaskTypeFollowingDownload, &api.FollowingDownloadTaskData{
			ScreenName: target,
		})
	default:
		task = b.taskManager.CreateTask(api.TaskTypeUserDownload, &api.UserDownloadTaskData{
			ScreenName: target,
		})
	}

	b.mu.Lock()
	if b.chatTasks[msg.Chat.ID] == nil {
		b.chatTasks[msg.Chat.ID] = make(map[string]struct{})
	}
	b.chatTasks[msg.Chat.ID][task.ID] = struct{}{}
	b.mu.Unlock()
	b.sendText(msg.Chat.ID, fmt.Sprintf("📥 Download started for %s\nTask: `%s`", target, task.ID))
}

func (b *Bot) cmdStatus(msg *tgbotapi.Message) {
	taskID := strings.TrimSpace(msg.CommandArguments())
	if taskID == "" {
		b.sendText(msg.Chat.ID, "Usage: /status <task_id>")
		return
	}
	task, ok := b.taskManager.GetTask(taskID)
	if !ok {
		b.sendText(msg.Chat.ID, "Task not found.")
		return
	}
	text := fmt.Sprintf("Task `%s`\nType: %s\nStatus: %s", task.ID, task.Type, task.Status)
	if task.Progress != nil {
		text += fmt.Sprintf("\nProgress: %d/%d (failed: %d)", task.Progress.Completed, task.Progress.Total, task.Progress.Failed)
	}
	if task.Error != "" {
		text += fmt.Sprintf("\nError: %s", task.Error)
	}
	b.sendText(msg.Chat.ID, text)
}

func (b *Bot) cmdCancel(msg *tgbotapi.Message) {
	taskID := strings.TrimSpace(msg.CommandArguments())
	if taskID == "" {
		b.sendText(msg.Chat.ID, "Usage: /cancel <task_id>")
		return
	}
	task, ok := b.taskManager.GetTask(taskID)
	if !ok {
		b.sendText(msg.Chat.ID, "Task not found.")
		return
	}
	if task.Status == api.TaskStatusCompleted || task.Status == api.TaskStatusFailed || task.Status == api.TaskStatusCancelled {
		b.sendText(msg.Chat.ID, "Task already in terminal state.")
		return
	}
	task.Cancel()
	b.sendText(msg.Chat.ID, fmt.Sprintf("Cancelling `%s`...", taskID))
}

func (b *Bot) cmdTasks(msg *tgbotapi.Message) {
	tasks := b.taskManager.GetAllTasks()
	if len(tasks) == 0 {
		b.sendText(msg.Chat.ID, "No tasks.")
		return
	}
	count := min(10, len(tasks))
	var sb strings.Builder
	for _, t := range tasks[:count] {
		sb.WriteString(fmt.Sprintf("`%s` | %s | %s\n", t.ID, t.Type, t.Status))
		if t.Error != "" {
			sb.WriteString(fmt.Sprintf("  ❌ %s\n", t.Error))
		}
	}
	b.sendText(msg.Chat.ID, sb.String())
}

func (b *Bot) cmdHelp(msg *tgbotapi.Message) {
	b.cmdStart(msg)
}

// parseDLArgs 解析 /dl 参数：支持 "user <name>", "list <id>", "foll <name>", 或裸 <name>
func parseDLArgs(raw string) (dlType, target string) {
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) == 1 {
		return "user", parts[0]
	}
	switch parts[0] {
	case "user", "list", "foll":
		return parts[0], strings.TrimSpace(parts[1])
	default:
		return "user", raw
	}
}
