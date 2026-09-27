package config

import (
	"flag"

	log "github.com/sirupsen/logrus"
)

// LoggingConfig defines logging for agent processes
type LoggingConfig struct {
	TextLogging    bool
	JournalLogging bool
	StdoutLogging  bool
}

// SubprocessConfig processe's logging configuration
type SubprocessConfig struct {
	LoggingConfig
	DryRunConfig
}

// RegisterLoggingArgs must not be called more than once per process.
// Subsequent calls will panic.
func RegisterLoggingArgs(loggingConfig *LoggingConfig) {
	flag.BoolVar(&loggingConfig.JournalLogging, "with-journal-logging", true, "Use journal logging")
	flag.BoolVar(&loggingConfig.TextLogging, "with-text-logging", false, "Use text logging")
	flag.BoolVar(&loggingConfig.StdoutLogging, "with-stdout-logging", false, "Use stdout logging")
}

// ProcessSubprocessArgs parses arguments
func ProcessSubprocessArgs() *SubprocessConfig {
	subprocessConfig := &SubprocessConfig{}

	RegisterLoggingArgs(&subprocessConfig.LoggingConfig)

	err := RegisterDryRunArgs(&subprocessConfig.DryRunConfig)
	if err != nil {
		log.Fatalf("Failed to register dry run arguments: %v", err)
	}

	h := flag.Bool("help", false, "Help message")
	flag.Parse()
	if h != nil && *h {
		printHelpAndExit()
	}

	return subprocessConfig
}
