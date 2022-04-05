package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/connectivity_check"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"
)

func main() {
	config.ProcessSubprocessArgs(config.DefaultLoggingConfig)
	config.ProcessDryRunArgs()
	util.SetLogging("connectivity-check", config.SubprocessConfig.TextLogging, config.SubprocessConfig.JournalLogging, config.GlobalDryRunConfig.ForcedHostID)
	if flag.NArg() != 1 {
		log.Warnf("Expecting exactly single argument to connectivity check. Received %d", len(os.Args)-1)
		os.Exit(-1)
	}
	stdout, stderr, exitCode := connectivity_check.ConnectivityCheck("", flag.Arg(0))
	fmt.Fprint(os.Stdout, stdout)
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(exitCode)
}
