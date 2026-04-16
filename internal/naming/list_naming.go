package naming

import (
	"github.com/unkmonster/tmd/internal/utils"
)

type ListNaming struct {
	baseNaming
}

func NewListNamingFromBase(lst interface{ GetId() int64; Title() string }) *ListNaming {
	return &ListNaming{
		baseNaming: baseNaming{
			sanitized: utils.WinFileNameWithMaxLen(lst.Title(), MaxFileNameLen),
		},
	}
}

func (ln *ListNaming) SanitizedTitle() string {
	return ln.sanitized
}
