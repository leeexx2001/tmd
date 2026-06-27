package bot

import (
	"strings"
)

// Bot 是 bot 平台的通用接口
type Bot interface {
	// Start 启动 bot。非阻塞，内部 goroutine 处理消息和事件。
	Start() error
	// Stop 停止 bot，等待 goroutine 退出。
	Stop()
	// Name 返回 bot 名称，用于日志和调试。
	Name() string
}

// DownloadOptions 下载选项，对应 api 层的 *TaskData 中的可选字段
type DownloadOptions struct {
	AutoFollow    bool
	FollowMembers bool
	SkipProfile   bool
	NoRetry       bool
}

// ParseDownloadOptions 从参数字符串末尾解析 key=value 选项并返回剩余参数。
// 支持: auto_follow/af, follow_members/fm, skip_profile/sp, no_retry/nr
// 值必须为 "true" 或 "false"。
// 示例: "elonmusk auto_follow=true skip_profile=true" → "elonmusk", {AutoFollow:true, SkipProfile:true}
func ParseDownloadOptions(raw string) (remaining string, opts DownloadOptions) {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return raw, opts
	}
	end := len(parts)
	for end > 0 {
		kv := strings.SplitN(parts[end-1], "=", 2)
		if len(kv) != 2 || (kv[1] != "true" && kv[1] != "false") {
			break
		}
		val := kv[1] == "true"
		switch kv[0] {
		case "auto_follow", "af":
			opts.AutoFollow = val
		case "follow_members", "fm":
			opts.FollowMembers = val
		case "skip_profile", "sp":
			opts.SkipProfile = val
		case "no_retry", "nr":
			opts.NoRetry = val
		}
		end--
	}
	return strings.Join(parts[:end], " "), opts
}

