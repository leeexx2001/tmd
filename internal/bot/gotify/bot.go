package gotify

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/api"
	"github.com/unkmonster/tmd/internal/config"
	"github.com/unkmonster/tmd/internal/consolelog"
)

// Bot 是 Gotify 推送 bot 实现（单向通知，无命令交互）
type Bot struct {
	config   *config.GotifyBotConfig
	eventBus *api.EventBus
	logHub   *consolelog.Hub
	client   *http.Client

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewBot 创建 Gotify bot 实例
func NewBot(cfg *config.GotifyBotConfig, eb *api.EventBus, lh *consolelog.Hub) *Bot {
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
	log.Infof("[bot-gotify] Started (server: %s)", b.config.ServerURL)
	return nil
}

// Stop 停止 bot。
func (b *Bot) Stop() {
	close(b.stopCh)
	b.wg.Wait()
}

// Name 返回 bot 名称
func (b *Bot) Name() string { return "gotify" }

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
		b.sendNotification(title, api.FormatTaskResult(task, true))
	}
}

func (b *Bot) sendLogAlert(line string) {
	b.sendNotification("🔴 TMD Error", line)
}

type gotifyMessage struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Priority int    `json:"priority"`
}

func (b *Bot) sendNotification(title, message string) {
	priority := b.config.Priority
	if priority == 0 {
		priority = 5
	}
	body, _ := json.Marshal(gotifyMessage{
		Title:    title,
		Message:  message,
		Priority: priority,
	})

	url := strings.TrimRight(b.config.ServerURL, "/") + "/message"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		log.Warnf("[bot-gotify] Failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gotify-Key", b.config.Token)

	resp, err := b.client.Do(req)
	if err != nil {
		log.Warnf("[bot-gotify] Failed to send: %v", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Warnf("[bot-gotify] Gotify returned %d", resp.StatusCode)
	}
}
