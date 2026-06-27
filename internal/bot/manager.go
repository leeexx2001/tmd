package bot

import (
	log "github.com/sirupsen/logrus"
)

// BotManager 管理多个 Bot 实例的生命周期
type BotManager struct {
	bots []Bot
}

// NewBotManager 创建 BotManager。可传入零个或多个 Bot 实现。
func NewBotManager(bots ...Bot) *BotManager {
	return &BotManager{bots: bots}
}

// Start 依次启动所有 bot。单个 bot 失败不影响其余 bot 启动。
func (bm *BotManager) Start() {
	for _, b := range bm.bots {
		if err := b.Start(); err != nil {
			log.Errorf("[bot] Failed to start %s: %v", b.Name(), err)
			continue
		}
		log.Infof("[bot] %s started", b.Name())
	}
}

// Stop 依次停止所有 bot。
func (bm *BotManager) Stop() {
	for _, b := range bm.bots {
		b.Stop()
		log.Infof("[bot] %s stopped", b.Name())
	}
}
