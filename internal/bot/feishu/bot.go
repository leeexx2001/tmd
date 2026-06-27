package feishu

import (
	"context"
	"net/http"
	"sync"

	"github.com/chyroc/lark"
	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/api"
	"github.com/unkmonster/tmd/internal/config"
	"github.com/unkmonster/tmd/internal/consolelog"
)

// Bot 是飞书/Lark bot 实现
type Bot struct {
	config      *config.FeishuBotConfig
	taskManager *api.TaskManager
	eventBus    *api.EventBus
	logHub      *consolelog.Hub

	cli *lark.Lark

	userChats map[string]string
	mu        sync.Mutex

	callbackHandler http.HandlerFunc

	stopCh chan struct{}
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// NewBot 创建飞书 bot 实例
func NewBot(cfg *config.FeishuBotConfig, tm *api.TaskManager, eb *api.EventBus, lh *consolelog.Hub) *Bot {
	return &Bot{
		config:      cfg,
		taskManager: tm,
		eventBus:    eb,
		logHub:      lh,
		userChats:   make(map[string]string),
		stopCh:      make(chan struct{}),
	}
}

// Start 启动 bot。初始化 Lark 客户端，注册事件回调，订阅事件/日志。
func (b *Bot) Start() error {
	encryptKey := b.config.EncryptKey
	opts := []lark.ClientOptionFunc{
		lark.WithAppCredential(b.config.AppID, b.config.AppSecret),
		lark.WithEventCallbackVerify(encryptKey, b.config.VerifyToken),
		lark.WithNonBlockingCallback(true),
	}

	cli := lark.New(opts...)
	b.cli = cli

	cli.EventCallback.HandlerEventV2IMMessageReceiveV1(func(ctx context.Context, cli *lark.Lark, schema string, header *lark.EventHeaderV2, event *lark.EventV2IMMessageReceiveV1) (string, error) {
		b.handleMessage(ctx, event)
		return "", nil
	})

	b.callbackHandler = func(w http.ResponseWriter, r *http.Request) {
		cli.EventCallback.ListenCallback(r.Context(), r.Body, w)
	}

	b.wg.Add(1)
	go b.handleEvents()
	if b.logHub != nil {
		b.wg.Add(1)
		go b.handleLogs()
	}

	log.Infof("[bot-feishu] Started (app_id: %s)", b.config.AppID)
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
func (b *Bot) Name() string { return "feishu" }

// CallbackHandler 返回 HTTP handler 供 Server 注册飞书事件回调路由
func (b *Bot) CallbackHandler() http.HandlerFunc {
	return b.callbackHandler
}

// CallbackPath 返回回调路径
func (b *Bot) CallbackPath() string {
	if b.config.CallbackPath != "" {
		return b.config.CallbackPath
	}
	return "/api/v1/bot/feishu/callback"
}

func (b *Bot) isAllowed(openID string) bool {
	if len(b.config.AllowedUsers) == 0 {
		return true
	}
	for _, id := range b.config.AllowedUsers {
		if id == openID {
			return true
		}
	}
	return false
}

func (b *Bot) sendText(chatID, text string) {
	ctx := context.Background()
	_, _, err := b.cli.Message.Send().ToChatID(chatID).SendText(ctx, text)
	if err != nil {
		log.Warnf("[bot-feishu] Failed to send message: %v", err)
	}
}

func (b *Bot) sendReply(msgID, text string) {
	ctx := context.Background()
	_, _, err := b.cli.Message.Reply(msgID).SendText(ctx, text)
	if err != nil {
		log.Warnf("[bot-feishu] Failed to reply: %v", err)
	}
}
