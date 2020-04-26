package util

import (
	"os"

	log "github.com/sirupsen/logrus"
)


func setLoggingOnLogger(name string, logger *log.Logger) {
	fname := "/var/log/" + name + ".log"
	file, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// We do not print since it is going to the input of the parent process
		return
	}
	logger.SetOutput(file)
	logger.SetFormatter(&log.TextFormatter{})

}

func SetLogging(name string) {
	setLoggingOnLogger(name, log.StandardLogger())
}
