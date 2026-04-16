package naming

import (
	"github.com/unkmonster/tmd/internal/utils"
)

const ExtReserveLen = 5

var MaxFileNameLen = utils.DefaultMaxFileNameLen

type baseNaming struct {
	sanitized string
}
