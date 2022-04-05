package actions

import (
	"github.com/openshift/assisted-installer-agent/src/domain_resolution"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

type domainResolution struct {
	args []string
}

func (a *domainResolution) Validate() error {
	modelToValidate := models.DomainResolutionRequest{}
	err := validateCommon("domain resolution", 1, a.args, &modelToValidate)
	if err != nil {
		return err
	}

	return nil
}

func (a *domainResolution) CreateCmd() (string, []string) {
	return "", nil
}

func (a *domainResolution) Run() (stdout, stderr string, exitCode int) {
	return domain_resolution.Run(a.args[0],
		&domain_resolution.DomainResolver{}, log.StandardLogger())
}

func (a *domainResolution) Command() string {
	return "domain_resolution"
}

func (a *domainResolution) Args() []string {
	return a.args
}