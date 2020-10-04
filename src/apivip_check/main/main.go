package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"

	"github.com/openshift/assisted-installer-agent/src/apivip_check"
	log "github.com/sirupsen/logrus"
)

func main() {
	config.ProcessSubprocessArgs(config.DefaultLoggingConfig)
	util.SetLogging("apivip_check", config.SubprocessConfig.TextLogging, config.SubprocessConfig.JournalLogging)
	if flag.NArg() != 1 {
		log.Warnf("Expecting exactly single argument to apivip_check. Received %d", len(os.Args)-1)
		os.Exit(-1)
	}
	stdout, stderr, exitCode := apivip_check.CheckAPIConnectivity(flag.Arg(0), log.StandardLogger())
	fmt.Fprint(os.Stdout, stdout)
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(exitCode)
}
