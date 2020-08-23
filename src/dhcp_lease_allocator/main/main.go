package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/dhcp_lease_allocator"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"
)

func main() {
	config.ProcessSubprocessArgs(false, true)
	util.SetLogging("dhcp_lease_allocator", config.SubprocessConfig.TextLogging, config.SubprocessConfig.JournalLogging)
	if flag.NArg() != 1 {
		log.Warnf("Expecting exactly single argument to dhcp_lease_allocator. Received %d", len(os.Args)-1)
		os.Exit(-1)
	}
	stdout, stderr, exitCode := dhcp_lease_allocator.LeaseAllocator(flag.Arg(0), &dhcp_lease_allocator.ProcessExecuter{}, log.StandardLogger())
	fmt.Fprint(os.Stdout, stdout)
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(exitCode)
}
