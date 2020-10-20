package config

import "flag"

var SubprocessConfig struct {
	TextLogging    bool
	JournalLogging bool
}

func ProcessSubprocessArgs(defaultTextLogging, defaultJournalLogging bool) {
	flag.BoolVar(&SubprocessConfig.JournalLogging, "with-journal-logging", defaultJournalLogging, "Use journal logging")
	flag.BoolVar(&SubprocessConfig.TextLogging, "with-text-logging", defaultTextLogging, "Use text logging")
	h := flag.Bool("help", false, "Help message")
	flag.Parse()
	if h != nil && *h {
		printHelpAndExit()
	}
}
