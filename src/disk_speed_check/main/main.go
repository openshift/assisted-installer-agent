package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/disk_speed_check"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"
)

func main() {
	config.ProcessSubprocessArgs(config.DefaultLoggingConfig)
	util.SetLogging("disk-speed-check", config.GlobalAgentConfig.TextLogging, config.GlobalAgentConfig.JournalLogging)

	req := flag.Arg(flag.NArg() - 1)
	perfCheck := disk_speed_check.NewDiskSpeedCheck(disk_speed_check.NewDependencies())
	stdout, stderr, exitCode := perfCheck.FioPerfCheck(req, log.StandardLogger())
	fmt.Fprint(os.Stdout, stdout)
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(exitCode)
}
