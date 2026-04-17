package utils

import (
	log "github.com/sirupsen/logrus"
)

func RecoverWithLog(name string) {
	if r := recover(); r != nil {
		log.Errorf("[%s] panic recovered: %v", name, r)
	}
}
