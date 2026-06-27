package discord

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/api"
)

func (b *Bot) cmdDownload(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	var dlType, target string
	for _, opt := range data.Options {
		switch opt.Name {
		case "type":
			dlType = opt.StringValue()
		case "target":
			target = opt.StringValue()
		}
	}
	if target == "" {
		b.respond(s, i, "Usage: /dl [type:user|list|foll] <target>")
		return
	}
	if dlType == "" {
		dlType = "user"
	}

	var task *api.Task
	switch dlType {
	case "list":
		listID, err := strconv.ParseUint(target, 10, 64)
		if err != nil {
			b.respond(s, i, "Invalid list ID. Must be a number.")
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

	channelID := i.ChannelID
	b.mu.Lock()
	if b.channelTasks[channelID] == nil {
		b.channelTasks[channelID] = make(map[string]struct{})
	}
	b.channelTasks[channelID][task.ID] = struct{}{}
	b.mu.Unlock()

	b.respond(s, i, fmt.Sprintf("📥 Download started for `%s`\nTask: `%s`", target, task.ID))
}

func (b *Bot) cmdStatus(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	options := data.Options
	if len(options) == 0 {
		b.respond(s, i, "Usage: /status <task_id>")
		return
	}
	taskID := options[0].StringValue()

	task, ok := b.taskManager.GetTask(taskID)
	if !ok {
		b.respond(s, i, "Task not found.")
		return
	}

	text := fmt.Sprintf("**%s**\nType: `%s`\nStatus: `%s`", task.ID, task.Type, task.Status)
	if task.Progress != nil {
		text += fmt.Sprintf("\nProgress: %d/%d (failed: %d)", task.Progress.Completed, task.Progress.Total, task.Progress.Failed)
	}
	if task.Error != "" {
		text += fmt.Sprintf("\nError: %s", task.Error)
	}
	b.respond(s, i, text)
}

func (b *Bot) cmdCancel(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	options := data.Options
	if len(options) == 0 {
		b.respond(s, i, "Usage: /cancel <task_id>")
		return
	}
	taskID := options[0].StringValue()

	task, ok := b.taskManager.GetTask(taskID)
	if !ok {
		b.respond(s, i, "Task not found.")
		return
	}
	if task.Status == api.TaskStatusCompleted || task.Status == api.TaskStatusFailed || task.Status == api.TaskStatusCancelled {
		b.respond(s, i, "Task already in terminal state.")
		return
	}
	task.Cancel()
	b.respond(s, i, fmt.Sprintf("Cancelling `%s`...", taskID))
}

func (b *Bot) cmdTasks(s *discordgo.Session, i *discordgo.InteractionCreate) {
	tasks := b.taskManager.GetAllTasks()
	if len(tasks) == 0 {
		b.respond(s, i, "No tasks.")
		return
	}

	count := min(10, len(tasks))
	text := "**Recent Tasks:**\n"
	for _, t := range tasks[:count] {
		text += fmt.Sprintf("`%s` | %s | %s\n", t.ID, t.Type, t.Status)
		if t.Error != "" {
			text += fmt.Sprintf("  ❌ %s\n", t.Error)
		}
	}
	b.respond(s, i, text)
}

func (b *Bot) cmdHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.respond(s, i, "**TMD Bot**\n/dl [type:user|list|foll] <target> — download\n/status <id> — task status\n/cancel <id> — cancel task\n/tasks — list recent tasks\n/help — this message")
}

func (b *Bot) respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	}); err != nil {
		log.Warnf("[bot-discord] Failed to respond: %v", err)
	}
}
