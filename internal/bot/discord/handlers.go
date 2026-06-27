package discord

import (
	"github.com/bwmarrin/discordgo"
)

// registerCommands 注册全局 slash commands
func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "dl",
			Description: "Download tweets",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "type",
					Description: "Download type: user, list, or foll",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "user", Value: "user"},
						{Name: "list", Value: "list"},
						{Name: "following", Value: "foll"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "target",
					Description: "Screen name or list ID",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "auto_follow",
				Description: "Auto-follow protected users (default false)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "skip_profile",
				Description: "Skip profile/avatar download (default false)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "no_retry",
				Description: "Skip retry on failure (default false)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "follow_members",
				Description: "Follow all list members (default false)",
				Required:    false,
			},
		},
	},
		{
			Name:        "status",
			Description: "Check task status",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "task_id",
					Description: "Task ID",
					Required:    true,
				},
			},
		},
		{
			Name:        "cancel",
			Description: "Cancel a running task",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "task_id",
					Description: "Task ID",
					Required:    true,
				},
			},
		},
		{
			Name:        "tasks",
			Description: "List recent tasks",
		},
		{
			Name:        "help",
			Description: "Show available commands",
		},
	}

	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

// handleInteraction 处理 slash command 交互
func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	if !b.isAllowed(userIDFromInteraction(i)) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⛔ Unauthorized",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	switch data.Name {
	case "dl":
		b.cmdDownload(s, i, data)
	case "status":
		b.cmdStatus(s, i, data)
	case "cancel":
		b.cmdCancel(s, i, data)
	case "tasks":
		b.cmdTasks(s, i)
	case "help":
		b.cmdHelp(s, i)
	}
}

// userIDFromInteraction 安全获取用户 ID，兼容频道和私聊
func userIDFromInteraction(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func (b *Bot) isAllowed(userID string) bool {
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
