package naming

import (
	"fmt"
	"path/filepath"

	"github.com/unkmonster/tmd/internal/utils"
)

// ExtReserveLen 为文件扩展名预留的长度（如 .json, .jpg 等）
const ExtReserveLen = 5

// MaxFileNameLen 使用 utils 包的默认值
var MaxFileNameLen = utils.DefaultMaxFileNameLen

func SetMaxFileNameLen(len int) {
	if len < 50 {
		len = 50
	}
	if len > 250 {
		len = 250
	}
	MaxFileNameLen = len
}

// baseNaming 包含名称处理的基础功能
type baseNaming struct {
	sanitized string
}

func (bn *baseNaming) Sanitized() string {
	return bn.sanitized
}

// TweetNaming 推文命名
type TweetNaming struct {
	baseNaming
	text    string
	tweetID uint64
	creator string
}

func NewTweetNaming(text string, tweetID uint64, creator string) *TweetNaming {
	return &TweetNaming{
		baseNaming: baseNaming{
			sanitized: utils.WinFileNameWithMaxLen(text, MaxFileNameLen),
		},
		text:    text,
		tweetID: tweetID,
		creator: creator,
	}
}

func (tn *TweetNaming) SanitizedText() string {
	return tn.sanitized
}

func (tn *TweetNaming) baseName() string {
	idPart := fmt.Sprintf("_%d", tn.tweetID)
	maxTextLen := MaxFileNameLen - len(idPart) - ExtReserveLen
	if maxTextLen < 0 {
		maxTextLen = 0
	}

	text := tn.sanitized
	if len(text) > maxTextLen {
		text = text[:maxTextLen]
	}
	if text == "" {
		text = "tweet"
	}

	return text + idPart
}

func (tn *TweetNaming) LogFormat() string {
	return fmt.Sprintf("[%s] %s", tn.creator, tn.baseName())
}

func (tn *TweetNaming) FilePrefix() string {
	return tn.baseName()
}

func (tn *TweetNaming) FileName(ext string) string {
	return tn.baseName() + ext
}

func (tn *TweetNaming) FilePath(dir string, ext string) (string, error) {
	fullPath := filepath.Join(dir, tn.FileName(ext))
	return utils.UniquePath(fullPath)
}

// UserNaming 用户命名
type UserNaming struct {
	baseNaming
	name       string
	screenName string
	title      string
}

func NewUserNaming(name, screenName string) *UserNaming {
	title := fmt.Sprintf("%s(%s)", name, screenName)
	return &UserNaming{
		baseNaming: baseNaming{
			sanitized: utils.WinFileNameWithMaxLen(title, MaxFileNameLen),
		},
		name:       name,
		screenName: screenName,
		title:      title,
	}
}

func (un *UserNaming) Title() string {
	return un.title
}

func (un *UserNaming) SanitizedTitle() string {
	return un.sanitized
}
