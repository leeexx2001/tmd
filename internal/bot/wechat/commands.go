package wechat

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/SpellingDragon/wechat-robot-go/wechat"

	"github.com/unkmonster/tmd/internal/api"
)

func (b *Bot) cmdDownload(ctx context.Context, msg *wechat.Message, args string) {
	if args == "" {
		b.wechatBot.Reply(ctx, msg, "Usage: /dl [user|list|foll] <target>\nDefaults to user if type omitted.")
		return
	}
	dlType, target := parseDLArgs(args)
	if target == "" {
		b.wechatBot.Reply(ctx, msg, "Usage: /dl [user|list|foll] <target>")
		return
	}

	var task *api.Task
	switch dlType {
	case "list":
		listID, err := strconv.ParseUint(target, 10, 64)
		if err != nil {
			b.wechatBot.Reply(ctx, msg, "Invalid list ID. Must be a number.")
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
	b.userTokens[msg.FromUserID] = msg.ContextToken
	b.mu.Unlock()

	b.wechatBot.Reply(ctx, msg, fmt.Sprintf("📥 Download started for %s\nTask: %s", target, task.ID))
}

func (b *Bot) cmdStatus(ctx context.Context, msg *wechat.Message, args string) {
	if args == "" {
		b.wechatBot.Reply(ctx, msg, "Usage: /status <task_id>")
		return
	}
	task, ok := b.taskManager.GetTask(args)
	if !ok {
		b.wechatBot.Reply(ctx, msg, "Task not found.")
		return
	}
	text := fmt.Sprintf("Task: %s\nType: %s\nStatus: %s", task.ID, task.Type, task.Status)
	if task.Progress != nil {
		text += fmt.Sprintf("\nProgress: %d/%d (failed: %d)", task.Progress.Completed, task.Progress.Total, task.Progress.Failed)
	}
	if task.Error != "" {
		text += fmt.Sprintf("\nError: %s", task.Error)
	}
	b.wechatBot.Reply(ctx, msg, text)
}

func (b *Bot) cmdCancel(ctx context.Context, msg *wechat.Message, args string) {
	if args == "" {
		b.wechatBot.Reply(ctx, msg, "Usage: /cancel <task_id>")
		return
	}
	task, ok := b.taskManager.GetTask(args)
	if !ok {
		b.wechatBot.Reply(ctx, msg, "Task not found.")
		return
	}
	if task.Status == api.TaskStatusCompleted || task.Status == api.TaskStatusFailed || task.Status == api.TaskStatusCancelled {
		b.wechatBot.Reply(ctx, msg, "Task already in terminal state.")
		return
	}
	task.Cancel()
	b.wechatBot.Reply(ctx, msg, fmt.Sprintf("Cancelling %s...", args))
}

func (b *Bot) cmdTasks(ctx context.Context, msg *wechat.Message) {
	tasks := b.taskManager.GetAllTasks()
	if len(tasks) == 0 {
		b.wechatBot.Reply(ctx, msg, "No tasks.")
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
	b.wechatBot.Reply(ctx, msg, text)
}

func (b *Bot) cmdHelp(ctx context.Context, msg *wechat.Message) {
	b.wechatBot.Reply(ctx, msg, "TMD Bot\n/dl [user|list|foll] <target> - download\n/status <id> - task status\n/cancel <id> - cancel task\n/tasks - list tasks\n/help - this message")
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
