package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/ssgreg/journald"
)


var getLogFileWriter = func (name string) (io.Writer, error) {
	fname := "/var/log/" + name + ".log"
	file, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// We do not print since it is going to the input of the parent process
		return nil, err
	}
	return file, nil
}

func setTextLogging(file io.Writer, logger *logrus.Logger) {
	logger.SetOutput(file)
}

func configureLogger(logger *logrus.Logger) {
	logger.SetReportCaller(true)
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat:        "02-01-2006 15:04:05", // the "time" field configuratiom
		FullTimestamp:          true,
		DisableLevelTruncation: true, // log level field configuration
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			// this function is required when you want to introduce your custom format.
			// In my case I wanted file and line to look like this `file="engine.go:141`
			// but f.File provides a full path along with the file name.
			// So in `formatFilePath()` function I just trimmet everything before the file name
			// and added a line number in the end
			return "", fmt.Sprintf("%s:%d", formatFilePath(f.File), f.Line)
		},
	})
}

func setNullWriter(logger *logrus.Logger) {
	logger.SetOutput(ioutil.Discard)
}

func formatFilePath(path string) string {
	arr := strings.Split(path, "/")
	return arr[len(arr)-1]
}

func setJournalLogging(logger *logrus.Logger, journalWriter IJournalWriter) {
	logger.AddHook(newJournalHook(journalWriter))
}

//go:generate mockery -name IJournalWriter -inpkg
type IJournalWriter interface {
	Send(msg string, p journald.Priority, fields map[string]interface{}) error
}

type journalHook struct{
	writer IJournalWriter
}

type JournalWriter struct {}

func (*JournalWriter) Send(msg string, p journald.Priority, fields map[string]interface{}) error {
	return journald.Send(msg, p, fields)
}

func newJournalHook(writer IJournalWriter) *journalHook {
	return &journalHook{writer:writer}
}

func (hook *journalHook) getPriority(entry *logrus.Entry) journald.Priority {
	switch entry.Level {
	case logrus.TraceLevel, logrus.DebugLevel:
		return journald.PriorityDebug
	case logrus.InfoLevel:
		return journald.PriorityInfo
	case logrus.WarnLevel:
		return journald.PriorityWarning
	case logrus.ErrorLevel:
		return journald.PriorityErr
	case logrus.FatalLevel:
		return journald.PriorityCrit
	case logrus.PanicLevel:
		return journald.PriorityEmerg
	default:
		return journald.PriorityInfo
	}
}

func (hook *journalHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	fields := map[string] interface{} {
		"TAG": "agent",
	}
	return hook.writer.Send(line, hook.getPriority(entry), fields)
}

func (hook *journalHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func setLogging(logger *logrus.Logger, journalWriter IJournalWriter, name string, textLogging, journalLogging bool) {
	configureLogger(logger)
	if textLogging {
		file, err := getLogFileWriter(name)
		if err == nil {
			setTextLogging(file, logger)
		}
	} else {
		setNullWriter(logger)
	}
	if journalLogging {
		setJournalLogging(logger, journalWriter)
	}
}

func SetLogging(name string, textLogging, journalLogging bool) {
	setLogging(logrus.StandardLogger(), &JournalWriter{}, name, textLogging, journalLogging)
}
