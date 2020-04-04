package util

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func SetLogging(name string) {
	fname := "/var/log/" + name + ".log"
	file, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// We do not print since it is going to the input of the parent process
		return
	}
	log.SetOutput(file)
	log.SetFormatter(&log.TextFormatter{})
}
