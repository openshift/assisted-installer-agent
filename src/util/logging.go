package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/openshift/assisted-installer-agent/pkg/journalLogger"
)

var getLogFileWriter = func(name string) (io.Writer, error) {
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

func setTextAndStdoutLogging(file io.Writer, logger *logrus.Logger) {
	mw := io.MultiWriter(os.Stdout, file)
	logger.SetOutput(mw)
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

func setLogging(logger *logrus.Logger, journalWriter journalLogger.IJournalWriter, name string, textLogging, journalLogging, stdOutLogging bool, hostID string) {
	configureLogger(logger)

	if textLogging {
		file, err := getLogFileWriter(name)
		if err == nil && !stdOutLogging {
			setTextLogging(file, logger)
		} else if err == nil {
			setTextAndStdoutLogging(file, logger)
		}
	} else if !stdOutLogging {
		setNullWriter(logger)
	}
	if journalLogging {
		journalLogger.SetJournalLogging(logger, journalWriter, map[string]interface{}{
			"TAG":          "agent",
			"DRY_AGENT_ID": hostID,
		})
	}
}

func SetLogging(name string, textLogging, journalLogging, stdoutLogging bool, hostID string) {
	setLogging(logrus.StandardLogger(), &journalLogger.JournalWriter{}, name, textLogging, journalLogging, stdoutLogging, hostID)
}

func NewJournalLogger(name string, hostID string) logrus.FieldLogger {
	log := logrus.New()
	setLogging(log, &journalLogger.JournalWriter{}, name, false, true, false, hostID)
	return log
}
