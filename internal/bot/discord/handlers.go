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
	if !b.isAllowed(i.Member.User.ID) {
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
