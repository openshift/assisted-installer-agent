package config

import "flag"

// LoggingConfig defines logging for agent processes
type LoggingConfig struct {
	TextLogging    bool
	JournalLogging bool
}

// SubprocessConfig processe's logging configuration
type SubprocessConfig struct {
	LoggingConfig
	DryRunConfig
}

// DefaultLoggingConfig pre-defined most commonly used defaults
var DefaultLoggingConfig = LoggingConfig{
	TextLogging:    false,
	JournalLogging: true,
}

// ProcessSubprocessArgs parses arguments
func ProcessSubprocessArgs(loggingDefaults LoggingConfig) *SubprocessConfig {
	subprocessConfig := &SubprocessConfig{}
	flag.BoolVar(&subprocessConfig.JournalLogging, "with-journal-logging", loggingDefaults.JournalLogging, "Use journal logging")
	flag.BoolVar(&subprocessConfig.TextLogging, "with-text-logging", loggingDefaults.TextLogging, "Use text logging")
	h := flag.Bool("help", false, "Help message")
	flag.Parse()
	if h != nil && *h {
		printHelpAndExit()
	}
	return subprocessConfig
}
