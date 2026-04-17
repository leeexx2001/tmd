package utils

import (
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

func CheckRespStatus(resp *resty.Response) error {
	if resp.StatusCode() >= 400 {
		return &HttpStatusError{Code: resp.StatusCode(), Msg: resp.String()}
	}
	return nil
}

type HttpStatusError struct {
	Code int
	Msg  string
}

func (err *HttpStatusError) Error() string {
	return fmt.Sprintf("HTTP Error: %d %s", err.Code, err.Msg)
}

func IsStatusCode(err error, code int) bool {
	e, ok := err.(*HttpStatusError)
	if !ok {
		return false
	}
	return e.Code == code
}

func StripAvatarSuffix(url string) string {
	url = strings.Replace(url, "_normal", "", 1)
	url = strings.Replace(url, "_bigger", "", 1)
	url = strings.Replace(url, "_mini", "", 1)
	return url
}
