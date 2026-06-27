package wechat

import (
	"context"
	"fmt"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/SpellingDragon/wechat-robot-go/wechat"

	"github.com/unkmonster/tmd/internal/api"
	"github.com/unkmonster/tmd/internal/config"
	"github.com/unkmonster/tmd/internal/consolelog"
)

// Bot 是微信 iLink bot 实现
type Bot struct {
	config      *config.WeChatBotConfig
	taskManager *api.TaskManager
	eventBus    *api.EventBus
	logHub      *consolelog.Hub

	wechatBot *wechat.Bot

	userTokens map[string]string
	mu         sync.Mutex

	stopCh chan struct{}
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// NewBot 创建微信 bot 实例
func NewBot(cfg *config.WeChatBotConfig, tm *api.TaskManager, eb *api.EventBus, lh *consolelog.Hub) *Bot {
	return &Bot{
		config:      cfg,
		taskManager: tm,
		eventBus:    eb,
		logHub:      lh,
		userTokens:  make(map[string]string),
		stopCh:      make(chan struct{}),
	}
}

// Start 启动 bot。扫码登录后订阅消息和事件。
func (b *Bot) Start() error {
	bot := wechat.NewBot(
		wechat.WithTokenFile(b.config.CredentialPath),
	)

	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel

	err := bot.Login(ctx, func(qrCode string) {
		log.Infof("[bot-wechat] QR code URL: %s", qrCode)
	})
	if err != nil {
		return fmt.Errorf("wechat: login failed: %w", err)
	}
	b.wechatBot = bot
	log.Infof("[bot-wechat] Logged in")

	bot.OnMessage(func(ctx context.Context, msg *wechat.Message) error {
		if !b.isAllowed(msg.FromUserID) {
			return bot.Reply(ctx, msg, "⛔ Unauthorized")
		}
		b.mu.Lock()
		b.userTokens[msg.FromUserID] = msg.ContextToken
		b.mu.Unlock()
		b.handleMessage(ctx, msg)
		return nil
	})

	b.wg.Add(2)
	go b.handleEvents()
	if b.logHub != nil {
		go b.handleLogs()
	}

	go func() {
		bot.Run(ctx)
	}()

	return nil
}

// Stop 停止 bot
func (b *Bot) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	close(b.stopCh)
	b.wg.Wait()
}

// Name 返回 bot 名称
func (b *Bot) Name() string { return "wechat" }

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

func (b *Bot) handleMessage(ctx context.Context, msg *wechat.Message) {
	text := strings.TrimSpace(msg.Text())
	if text == "" || !strings.HasPrefix(text, "/") {
		return
	}
	parts := strings.SplitN(text, " ", 2)
	cmd := strings.TrimPrefix(parts[0], "/")
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}
	switch cmd {
	case "dl", "download":
		b.cmdDownload(ctx, msg, args)
	case "status":
		b.cmdStatus(ctx, msg, args)
	case "cancel":
		b.cmdCancel(ctx, msg, args)
	case "tasks":
		b.cmdTasks(ctx, msg)
	case "help":
		b.cmdHelp(ctx, msg)
	default:
		b.wechatBot.Reply(ctx, msg, "Unknown command. Send /help for available commands.")
	}
}

func (b *Bot) sendText(ctx context.Context, userID, text string) {
	if err := b.wechatBot.SendTextToUser(ctx, userID, text); err != nil {
		log.Warnf("[bot-wechat] Failed to send message to %s: %v", userID, err)
	}
}
