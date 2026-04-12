package naming

import (
	"fmt"
	"path/filepath"

	"github.com/unkmonster/tmd/internal/utils"
)

const DefaultMaxFileNameLen = 155

var MaxFileNameLen = DefaultMaxFileNameLen

func SetMaxFileNameLen(len int) {
	if len < 50 {
		len = 50
	}
	if len > 250 {
		len = 250
	}
	MaxFileNameLen = len
}

type TweetNaming struct {
	text      string
	tweetID   uint64
	creator   string
	sanitized string
}

func NewTweetNaming(text string, tweetID uint64, creator string) *TweetNaming {
	tn := &TweetNaming{
		text:    text,
		tweetID: tweetID,
		creator: creator,
	}
	tn.sanitized = utils.WinFileNameWithMaxLen(text, MaxFileNameLen)
	return tn
}

func (tn *TweetNaming) SanitizedText() string {
	return tn.sanitized
}

func (tn *TweetNaming) LogFormat() string {
	return fmt.Sprintf("[%s] %s_%d", tn.creator, tn.sanitized, tn.tweetID)
}

func (tn *TweetNaming) FilePrefix() string {
	idPart := fmt.Sprintf("_%d", tn.tweetID)
	maxTextLen := MaxFileNameLen - len(idPart) - 5
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

func (tn *TweetNaming) FileName(ext string) string {
	return tn.FilePrefix() + ext
}

func (tn *TweetNaming) FilePath(dir string, ext string) (string, error) {
	fileName := tn.FileName(ext)
	fullPath := filepath.Join(dir, fileName)
	return utils.UniquePath(fullPath)
}

type UserNaming struct {
	name       string
	screenName string
	title      string
	sanitized  string
}

func NewUserNaming(name, screenName string) *UserNaming {
	un := &UserNaming{
		name:       name,
		screenName: screenName,
	}
	un.title = fmt.Sprintf("%s(%s)", name, screenName)
	un.sanitized = utils.WinFileNameWithMaxLen(un.title, MaxFileNameLen)
	return un
}

func (un *UserNaming) Title() string {
	return un.title
}

func (un *UserNaming) SanitizedTitle() string {
	return un.sanitized
}
