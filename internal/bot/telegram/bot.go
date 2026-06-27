package telegram

import (
	"fmt"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/api"
	"github.com/unkmonster/tmd/internal/config"
	"github.com/unkmonster/tmd/internal/consolelog"
)

// Bot 是 Telegram bot 实现
type Bot struct {
	config      *config.TelegramBotConfig
	taskManager *api.TaskManager
	eventBus    *api.EventBus
	logHub      *consolelog.Hub

	api     *tgbotapi.BotAPI
	updates tgbotapi.UpdatesChannel

	chatTasks map[int64]map[string]struct{}
	mu        sync.Mutex

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewBot 创建 Telegram bot 实例
func NewBot(cfg *config.TelegramBotConfig, tm *api.TaskManager, eb *api.EventBus, lh *consolelog.Hub) *Bot {
	return &Bot{
		config:      cfg,
		taskManager: tm,
		eventBus:    eb,
		logHub:      lh,
		chatTasks:   make(map[int64]map[string]struct{}),
		stopCh:      make(chan struct{}),
	}
}

// Start 启动 bot。非阻塞，启动三个后台 goroutine 处理消息、事件和日志。
func (b *Bot) Start() error {
	botAPI, err := tgbotapi.NewBotAPI(b.config.Token)
	if err != nil {
		return fmt.Errorf("telegram: failed to create bot: %w", err)
	}
	b.api = botAPI
	log.Infof("[bot-telegram] Authorized as %s", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	b.updates = b.api.GetUpdatesChan(u)

	b.wg.Add(1)
	go b.handleUpdates()
	b.wg.Add(1)
	go b.handleEvents()
	if b.logHub != nil {
		b.wg.Add(1)
		go b.handleLogs()
	}
	return nil
}

// Stop 停止 bot，等待所有 goroutine 退出。
func (b *Bot) Stop() {
	close(b.stopCh)
	b.wg.Wait()
	if b.api != nil {
		b.api.StopReceivingUpdates()
	}
}

// Name 返回 bot 名称
func (b *Bot) Name() string { return "telegram" }

// sendText 发送 Markdown 文本消息到指定 chat
func (b *Bot) sendText(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "markdown"
	if _, err := b.api.Send(msg); err != nil {
		log.Warnf("[bot-telegram] Failed to send message: %v", err)
	}
}
