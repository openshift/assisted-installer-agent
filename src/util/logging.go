package util

import (
	"context"
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

const (
	loggerKey = "logger-key"
)

func WithLogger(ctx context.Context, logger log.FieldLogger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func ToLogger(ctx context.Context) log.FieldLogger {
	logger := ctx.Value(loggerKey)
	if logger == nil {
		logger = log.New()
		log.Warn("Did not found logger in context")
	}
	return logger.(log.FieldLogger)
}
