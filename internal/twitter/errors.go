package twitter

import (
	"fmt"

	"github.com/tidwall/gjson"
)

const (
	ErrTimeout         = 29
	ErrDependency      = 0
	ErrExceedPostLimit = 88
	ErrOverCapacity    = 130
	ErrAccountLocked   = 326
)

func CheckApiResp(body []byte) error {
	errors := gjson.GetBytes(body, "errors")
	if !errors.Exists() {
		return nil
	}

	codej := errors.Get("0.extensions.code")
	code := -1
	if codej.Exists() {
		code = int(codej.Int())
	}

	hasData := gjson.GetBytes(body, "data").Exists()
	if hasData {
		if code == 214 {
			return nil
		}
	}

	return NewTwitterApiError(code, string(body))
}

type TwitterApiError struct {
	Code int
	raw  string
}

func (err *TwitterApiError) Error() string {
	return fmt.Sprintf("Twitter API error (code %d): %s", err.Code, err.getMessage())
}

func (err *TwitterApiError) getMessage() string {
	errors := gjson.Get(err.raw, "errors")
	if errors.Exists() && len(errors.Array()) > 0 {
		msg := errors.Get("0.message").String()
		if msg != "" {
			return msg
		}
	}
	return "unknown error"
}

func NewTwitterApiError(code int, raw string) *TwitterApiError {
	return &TwitterApiError{Code: code, raw: raw}
}
