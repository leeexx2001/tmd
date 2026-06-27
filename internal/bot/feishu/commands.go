package feishu

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/unkmonster/tmd/internal/api"
)

func (b *Bot) cmdDownload(msgID, raw string) {
	if raw == "" {
		b.sendReply(msgID, "Usage: /dl [user|list|foll] <target>")
		return
	}
	dlType, target := parseDLArgs(raw)
	if target == "" {
		b.sendReply(msgID, "Usage: /dl [user|list|foll] <target>")
		return
	}

	var task *api.Task
	switch dlType {
	case "list":
		listID, err := strconv.ParseUint(target, 10, 64)
		if err != nil {
			b.sendReply(msgID, "Invalid list ID. Must be a number.")
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

	b.sendReply(msgID, fmt.Sprintf("📥 Download started for %s\nTask: %s", target, task.ID))
}

func (b *Bot) cmdStatus(msgID, taskID string) {
	if taskID == "" {
		b.sendReply(msgID, "Usage: /status <task_id>")
		return
	}
	task, ok := b.taskManager.GetTask(taskID)
	if !ok {
		b.sendReply(msgID, "Task not found.")
		return
	}
	text := fmt.Sprintf("Task: %s\nType: %s\nStatus: %s", task.ID, task.Type, task.Status)
	if task.Progress != nil {
		text += fmt.Sprintf("\nProgress: %d/%d (failed: %d)", task.Progress.Completed, task.Progress.Total, task.Progress.Failed)
	}
	if task.Error != "" {
		text += fmt.Sprintf("\nError: %s", task.Error)
	}
	b.sendReply(msgID, text)
}

func (b *Bot) cmdCancel(msgID, taskID string) {
	if taskID == "" {
		b.sendReply(msgID, "Usage: /cancel <task_id>")
		return
	}
	task, ok := b.taskManager.GetTask(taskID)
	if !ok {
		b.sendReply(msgID, "Task not found.")
		return
	}
	if task.Status == api.TaskStatusCompleted || task.Status == api.TaskStatusFailed || task.Status == api.TaskStatusCancelled {
		b.sendReply(msgID, "Task already in terminal state.")
		return
	}
	task.Cancel()
	b.sendReply(msgID, fmt.Sprintf("Cancelling %s...", taskID))
}

func (b *Bot) cmdTasks(msgID string) {
	tasks := b.taskManager.GetAllTasks()
	if len(tasks) == 0 {
		b.sendReply(msgID, "No tasks.")
		return
	}
	count := min(10, len(tasks))
	text := "Recent Tasks:\n"
	for _, t := range tasks[:count] {
		text += fmt.Sprintf("%s | %s | %s\n", t.ID, t.Type, t.Status)
		if t.Error != "" {
			text += fmt.Sprintf("  Error: %s\n", t.Error)
		}
	}
	b.sendReply(msgID, text)
}

func (b *Bot) cmdHelp(msgID string) {
	b.sendReply(msgID, "TMD Bot\n/dl [user|list|foll] <target> - download\n/status <id> - task status\n/cancel <id> - cancel task\n/tasks - list tasks\n/help - this message")
}

// parseDLArgs 解析 /dl 参数
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
