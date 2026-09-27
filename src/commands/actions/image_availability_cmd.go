package actions

import (
	"github.com/openshift/assisted-installer-agent/src/config"
	log "github.com/sirupsen/logrus"

	"github.com/openshift/assisted-installer-agent/src/container_image_availability"
	"github.com/openshift/assisted-service/models"
	"golang.org/x/sync/semaphore"
)

var sem = semaphore.NewWeighted(1)

type imageAvailability struct {
	args        []string
	agentConfig *config.AgentConfig
}

func (a *imageAvailability) Validate() error {
	modelToValidate := models.ContainerImageAvailabilityRequest{}
	err := ValidateCommon("image availability", 1, a.args, &modelToValidate)
	if err != nil {
		return err
	}
	return nil
}

func (a *imageAvailability) Run() (stdout, stderr string, exitCode int) {
	if !sem.TryAcquire(1) {
		log.Infof("%s already running", a.Command())
		return "", "", 0
	}
	defer sem.Release(1)
	subprocessConfig := &config.SubprocessConfig{LoggingConfig: a.agentConfig.LoggingConfig,
		DryRunConfig: a.agentConfig.DryRunConfig}

	return container_image_availability.Run(subprocessConfig, a.Args()[0],
		&container_image_availability.ProcessExecuter{}, log.StandardLogger())
}

func (a *imageAvailability) Command() string {
	return "container_image_availability"
}

func (a *imageAvailability) Args() []string {
	return a.args
}
