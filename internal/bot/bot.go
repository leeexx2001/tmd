package bot

// Bot 是 bot 平台的通用接口
type Bot interface {
	// Start 启动 bot。非阻塞，内部 goroutine 处理消息和事件。
	Start() error
	// Stop 停止 bot，等待 goroutine 退出。
	Stop()
	// Name 返回 bot 名称，用于日志和调试。
	Name() string
}
