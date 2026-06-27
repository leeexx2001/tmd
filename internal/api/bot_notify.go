package api

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/unkmonster/tmd/internal/consolelog"
)

// RunBotEventLoop 启动事件订阅协程：监听 EventBus，筛选 "tasks" 事件并回调。
// 所有 bot 平台的 handleEvents() 均替换为此函数。
func RunBotEventLoop(eb *EventBus, stopCh <-chan struct{}, wg *sync.WaitGroup, fn func(evt SSEEvent)) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		ch, unsub := eb.Subscribe()
		defer unsub()
		for {
			select {
			case <-stopCh:
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}
				if evt.Event != "tasks" {
					continue
				}
				fn(evt)
			}
		}
	}()
}

// RunBotLogLoop 启动日志订阅协程：监听 LogHub，筛选 error/fatal 级别日志，
// 以 1 秒速率限制回调。所有 bot 平台的 handleLogs() 均替换为此函数。
func RunBotLogLoop(lh *consolelog.Hub, stopCh <-chan struct{}, wg *sync.WaitGroup, fn func(line string)) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		ch, unsub := lh.Subscribe()
		defer unsub()
		var lastLog time.Time
		for {
			select {
			case <-stopCh:
				return
			case line, ok := <-ch:
				if !ok {
					return
				}
				if !strings.Contains(line, "level=error") && !strings.Contains(line, "level=fatal") {
					continue
				}
				now := time.Now()
				if now.Sub(lastLog) < time.Second {
					continue
				}
				lastLog = now
				fn(line)
			}
		}
	}()
}

// FormatTaskResult 格式化任务结果描述。
// markdown=true 时任务 ID 用反引号包裹（用于 Telegram、Discord）。
func FormatTaskResult(task *Task, markdown bool) string {
	icon := "✅"
	if task.Status == TaskStatusFailed {
		icon = "❌"
	}
	id := task.ID
	if markdown {
		id = "`" + id + "`"
	}
	text := fmt.Sprintf("%s Task %s %s", icon, id, task.Status)
	if task.Result != nil && task.Result.Main != nil {
		text += fmt.Sprintf("\nDownloaded: %d, Failed: %d", task.Result.Main.Downloaded, task.Result.Main.Failed)
	}
	if task.Error != "" {
		text += fmt.Sprintf("\nError: %s", task.Error)
	}
	return text
}
