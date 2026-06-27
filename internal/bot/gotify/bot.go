package gotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/api"
	"github.com/unkmonster/tmd/internal/bot"
	"github.com/unkmonster/tmd/internal/config"
	"github.com/unkmonster/tmd/internal/consolelog"
)

// Bot 是 Gotify 推送 bot 实现（单向通知，无命令交互）
type Bot struct {
	config   *config.GotifyBotConfig
	eventBus *api.EventBus
	logHub   *consolelog.Hub
	client   *http.Client

	stopCh      chan struct{}
	wg          sync.WaitGroup
	logThrottle bot.LogThrottle
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

// Start 启动 bot。非阻塞，订阅 EventBus 和 LogHub。
func (b *Bot) Start() error {
	b.wg.Add(1)
	go b.handleEvents()
	if b.logHub != nil {
		b.wg.Add(1)
		go b.handleLogs()
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

func (b *Bot) handleEvents() {
	defer b.wg.Done()
	ch, unsub := b.eventBus.Subscribe()
	defer unsub()
	for {
		select {
		case <-b.stopCh:
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			if evt.Event != "tasks" {
				continue
			}
			b.notifyTaskChanges(evt.Data)
		}
	}
}

func (b *Bot) notifyTaskChanges(data interface{}) {
	tasks, ok := data.([]*api.Task)
	if !ok {
		return
	}
	for _, task := range tasks {
		if task.Status != api.TaskStatusCompleted && task.Status != api.TaskStatusFailed {
			continue
		}
		b.sendNotification(b.formatTaskResult(task))
	}
}

func (b *Bot) formatTaskResult(task *api.Task) (title, message string) {
	if task.Status == api.TaskStatusCompleted {
		title = "✅ TMD Download Complete"
	} else {
		title = "❌ TMD Download Failed"
	}
	message = fmt.Sprintf("Task `%s` %s", task.ID, task.Status)
	if task.Result != nil && task.Result.Main != nil {
		message += fmt.Sprintf("\nDownloaded: %d, Failed: %d", task.Result.Main.Downloaded, task.Result.Main.Failed)
	}
	if task.Error != "" {
		message += fmt.Sprintf("\nError: %s", task.Error)
	}
	return
}

func (b *Bot) handleLogs() {
	defer b.wg.Done()
	ch, unsub := b.logHub.Subscribe()
	defer unsub()
	for {
		select {
		case <-b.stopCh:
			return
		case line, ok := <-ch:
			if !ok {
				return
			}
			if !strings.Contains(line, "level=error") && !strings.Contains(line, "level=fatal") {
				continue
			}
			if !b.logThrottle.Allow() {
				continue
			}
			b.sendNotification("🔴 TMD Error", line)
		}
	}
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
