package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/container_image_availability"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Request string
}

var executableConfig Config

func processArgs() {
	ret := &executableConfig
	flag.StringVar(&ret.Request, "request", "", "The request details. See models.ContainerImageAvailabilityRequest")

	flag.Parse()

	if executableConfig.Request == "" {
		flag.CommandLine.Usage()
		os.Exit(1)
	}
}

func main() {
	processArgs()
	config.ProcessSubprocessArgs(config.DefaultLoggingConfig)
	config.ProcessDryRunArgs()
	util.SetLogging("container_image_availability", config.SubprocessConfig.TextLogging, config.SubprocessConfig.JournalLogging, config.GlobalDryRunConfig.ForcedHostID)
	log.StandardLogger().Infof("Checking image availability, requested images: %s", executableConfig.Request)
	stdout, stderr, exitCode := container_image_availability.Run(executableConfig.Request,
		&container_image_availability.ProcessExecuter{}, log.StandardLogger())
	fmt.Fprint(os.Stdout, stdout)
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(exitCode)
}
