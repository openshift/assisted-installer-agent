package free_addresses

import (
	"flag"
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"

	log "github.com/sirupsen/logrus"
)

func Main() {
	subprocessConfig := config.ProcessSubprocessArgs(config.DefaultLoggingConfig)
	config.ProcessDryRunArgs(&subprocessConfig.DryRunConfig)

	util.SetLogging("free_addresses", subprocessConfig.TextLogging, subprocessConfig.JournalLogging, subprocessConfig.StdoutLogging, subprocessConfig.ForcedHostID)
	if flag.NArg() != 1 {
		log.Warnf("Expecting exactly single argument to free_addresses. Received %d", len(os.Args)-1)
		os.Exit(-1)
	}

	stdout, stderr, exitCode := GetFreeAddresses(flag.Arg(0), &ProcessExecuter{}, log.StandardLogger(), subprocessConfig.DryRunEnabled)
	fmt.Fprint(os.Stdout, stdout)
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(exitCode)
}
