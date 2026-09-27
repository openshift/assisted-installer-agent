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

var request string

func main() {
	flag.StringVar(&request, "request", "", "The request details. See models.ContainerImageAvailabilityRequest")

	subprocessConfig := config.ProcessSubprocessArgs()

	if request == "" {
		flag.CommandLine.Usage()
		os.Exit(1)
	}

	util.SetLogging("container_image_availability", subprocessConfig.TextLogging, subprocessConfig.JournalLogging, subprocessConfig.StdoutLogging, subprocessConfig.ForcedHostID)
	log.StandardLogger().Infof("Checking image availability, requested images: %s", request)
	stdout, stderr, exitCode := container_image_availability.Run(subprocessConfig, request,
		&container_image_availability.ProcessExecuter{}, log.StandardLogger())
	fmt.Fprint(os.Stdout, stdout)
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(exitCode)
}
