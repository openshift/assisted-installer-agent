package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/ntp_synchronizer"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"
)

func DryRunNtp() (string, string, int) {
	return `{"ntp_sources": []}`, "", 0
}

func main() {
	config.ProcessSubprocessArgs(config.DefaultLoggingConfig)
	config.ProcessDryRunArgs()
	util.SetLogging("ntp_synchronizer", config.SubprocessConfig.TextLogging, config.SubprocessConfig.JournalLogging, config.GlobalDryRunConfig.ForcedHostID)
	if flag.NArg() != 1 {
		log.Fatalf("Expecting exactly single argument to ntp_synchronizer. Received %d", len(os.Args)-1)
	}

	// Skip NTP in dry run mode, it's too expensive
	stdout, stderr, exitCode := DryRunNtp()
	if !config.GlobalDryRunConfig.DryRunEnabled {
		stdout, stderr, exitCode = ntp_synchronizer.Run(flag.Arg(0), &ntp_synchronizer.ProcessExecuter{}, log.StandardLogger())
	}

	fmt.Fprint(os.Stdout, stdout)
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(exitCode)
}
