package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/dhcp_lease_allocate"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"
)

func main() {
	subprocessConfig := config.ProcessSubprocessArgs(config.DefaultLoggingConfig)
	config.ProcessDryRunArgs(&subprocessConfig.DryRunConfig)
	util.SetLogging("dhcp_lease_allocate", subprocessConfig.TextLogging, subprocessConfig.JournalLogging, subprocessConfig.StdoutLogging, subprocessConfig.ForcedHostID)
	if flag.NArg() != 1 {
		log.Warnf("Expecting exactly single argument to dhcp_lease_allocate. Received %d", len(os.Args)-1)
		os.Exit(-1)
	}
	leaser := dhcp_lease_allocate.NewLeaser(dhcp_lease_allocate.NewLeaserDependencies())
	stdout, stderr, exitCode := leaser.LeaseAllocate(flag.Arg(0), log.StandardLogger())
	fmt.Fprint(os.Stdout, stdout)
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(exitCode)
}
