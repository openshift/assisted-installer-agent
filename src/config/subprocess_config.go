package config

import "flag"

// LoggingConfig defines logging for agent processes
type LoggingConfig struct {
	TextLogging    bool
	JournalLogging bool
}

// SubprocessConfig processe's logging configuration
var SubprocessConfig LoggingConfig

// DefaultLoggingConfig pre-defined most commonly used defaults
var DefaultLoggingConfig LoggingConfig = LoggingConfig{
	TextLogging:    false,
	JournalLogging: true,
}

// ProcessSubprocessArgs parses arguments
func ProcessSubprocessArgs(loggingDefaults LoggingConfig) {
	flag.BoolVar(&SubprocessConfig.JournalLogging, "with-journal-logging", loggingDefaults.JournalLogging, "Use journal logging")
	flag.BoolVar(&SubprocessConfig.TextLogging, "with-text-logging", loggingDefaults.TextLogging, "Use text logging")
	h := flag.Bool("help", false, "Help message")
	flag.Parse()
	if h != nil && *h {
		printHelpAndExit()
	}
}
