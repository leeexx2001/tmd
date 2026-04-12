package utils

import (
	"fmt"
	"runtime"

	log "github.com/sirupsen/logrus"
)

type PanicHandler struct {
	Name       string
	OnPanic    func(any)
	StackTrace bool
}

func (h *PanicHandler) Recover() bool {
	if r := recover(); r != nil {
		log.Errorf("[%s] panic recovered: %v", h.Name, r)

		if h.StackTrace {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			log.Debugf("[%s] stack trace:\n%s", h.Name, buf[:n])
		}

		if h.OnPanic != nil {
			h.OnPanic(r)
		}
		return true
	}
	return false
}

func RecoverWithLog(name string) {
	if r := recover(); r != nil {
		log.Errorf("[%s] panic recovered: %v", name, r)
	}
}

func RecoverWithError(name string, err *error) {
	if r := recover(); r != nil {
		log.Errorf("[%s] panic recovered: %v", name, r)
		*err = fmt.Errorf("[%s] panic: %v", name, r)
	}
}
