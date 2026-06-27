package pushover

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/api"
	"github.com/unkmonster/tmd/internal/config"
	"github.com/unkmonster/tmd/internal/consolelog"
)

// Bot 是 Pushover 推送 bot 实现（单向通知，无命令交互）
type Bot struct {
	config   *config.PushoverBotConfig
	eventBus *api.EventBus
	logHub   *consolelog.Hub
	client   *http.Client

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewBot 创建 Pushover bot 实例
func NewBot(cfg *config.PushoverBotConfig, eb *api.EventBus, lh *consolelog.Hub) *Bot {
	return &Bot{
		config:   cfg,
		eventBus: eb,
		logHub:   lh,
		client:   &http.Client{Timeout: 10 * time.Second},
		stopCh:   make(chan struct{}),
	}
}

func (b *Bot) Start() error {
	api.RunBotEventLoop(b.eventBus, b.stopCh, &b.wg, func(evt api.SSEEvent) {
		b.notifyTaskChanges(evt.Data)
	})
	if b.logHub != nil {
		api.RunBotLogLoop(b.logHub, b.stopCh, &b.wg, b.sendLogAlert)
	}
	log.Infof("[bot-pushover] Started")
	return nil
}

// Stop 停止 bot
func (b *Bot) Stop() {
	close(b.stopCh)
	b.wg.Wait()
}

// Name 返回 bot 名称
func (b *Bot) Name() string { return "pushover" }

func (b *Bot) notifyTaskChanges(data interface{}) {
	tasks, ok := data.([]*api.Task)
	if !ok {
		return
	}
	for _, task := range tasks {
		if task.Status != api.TaskStatusCompleted && task.Status != api.TaskStatusFailed {
			continue
		}
		title := "❌ TMD Download Failed"
		if task.Status == api.TaskStatusCompleted {
			title = "✅ TMD Download Complete"
		}
		b.sendNotification(title, api.FormatTaskResult(task, false))
	}
}

func (b *Bot) sendLogAlert(line string) {
	b.sendNotification("🔴 TMD Error", line)
}

func (b *Bot) sendNotification(title, message string) {
	vals := url.Values{
		"token":   {b.config.Token},
		"user":    {b.config.User},
		"title":   {title},
		"message": {message},
	}
	if b.config.Device != "" {
		vals.Set("device", b.config.Device)
	}
	if b.config.Sound != "" {
		vals.Set("sound", b.config.Sound)
	}

	resp, err := b.client.PostForm("https://api.pushover.net/1/messages.json", vals)
	if err != nil {
		log.Warnf("[bot-pushover] Failed to send notification: %v", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Warnf("[bot-pushover] Pushover returned %d", resp.StatusCode)
	}
}
