package pushover

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/unkmonster/tmd/internal/api"
	"github.com/unkmonster/tmd/internal/bot"
	"github.com/unkmonster/tmd/internal/config"
	"github.com/unkmonster/tmd/internal/consolelog"
)

// Bot 是 Pushover 推送 bot 实现（单向通知，无命令交互）
type Bot struct {
	config   *config.PushoverBotConfig
	eventBus *api.EventBus
	logHub   *consolelog.Hub
	client   *http.Client

	stopCh      chan struct{}
	wg          sync.WaitGroup
	logThrottle bot.LogThrottle
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

// Start 启动 bot
func (b *Bot) Start() error {
	b.wg.Add(1)
	go b.handleEvents()
	if b.logHub != nil {
		b.wg.Add(1)
		go b.handleLogs()
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
	message = fmt.Sprintf("Task %s %s", task.ID, task.Status)
	if task.Result != nil && task.Result.Main != nil {
		message += fmt.Sprintf(" | D:%d F:%d", task.Result.Main.Downloaded, task.Result.Main.Failed)
	}
	if task.Error != "" {
		message += fmt.Sprintf(" | Error: %s", task.Error)
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
