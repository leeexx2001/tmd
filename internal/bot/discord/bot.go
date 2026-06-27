package discord

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/api"
	"github.com/unkmonster/tmd/internal/config"
	"github.com/unkmonster/tmd/internal/consolelog"
)

// Bot 是 Discord bot 实现
type Bot struct {
	config      *config.DiscordBotConfig
	taskManager *api.TaskManager
	eventBus    *api.EventBus
	logHub      *consolelog.Hub

	session *discordgo.Session

	channelTasks map[string]map[string]struct{}
	mu           sync.Mutex

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewBot 创建 Discord bot 实例
func NewBot(cfg *config.DiscordBotConfig, tm *api.TaskManager, eb *api.EventBus, lh *consolelog.Hub) *Bot {
	return &Bot{
		config:       cfg,
		taskManager:  tm,
		eventBus:     eb,
		logHub:       lh,
		channelTasks: make(map[string]map[string]struct{}),
		stopCh:       make(chan struct{}),
	}
}

// Start 启动 bot。非阻塞，使用 discordgo Gateway 连接。
func (b *Bot) Start() error {
	session, err := discordgo.New("Bot " + b.config.Token)
	if err != nil {
		return fmt.Errorf("discord: failed to create session: %w", err)
	}
	b.session = session

	session.AddHandler(b.handleInteraction)

	session.Identify.Intents = discordgo.IntentsGuildMessages

	if err := session.Open(); err != nil {
		return fmt.Errorf("discord: failed to open gateway: %w", err)
	}
	log.Infof("[bot-discord] Connected as %s", session.State.User.Username)

	if err := b.registerCommands(); err != nil {
		log.Warnf("[bot-discord] Failed to register slash commands: %v", err)
	}

	api.RunBotEventLoop(b.eventBus, b.stopCh, &b.wg, func(evt api.SSEEvent) {
		b.notifyTaskChanges(evt.Data)
	})
	if b.logHub != nil {
		api.RunBotLogLoop(b.logHub, b.stopCh, &b.wg, b.sendLogAlert)
	}
	return nil
}

// Stop 停止 bot，关闭 Gateway 连接。
func (b *Bot) Stop() {
	close(b.stopCh)
	b.wg.Wait()
	if b.session != nil {
		if err := b.session.Close(); err != nil {
			log.Warnf("[bot-discord] Session close error: %v", err)
		}
	}
}
// Name 返回 bot 名称
func (b *Bot) Name() string { return "discord" }
