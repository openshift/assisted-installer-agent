package actions

import (
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/vips_verifier"
	"github.com/openshift/assisted-service/models"
)

type vipsVerifier struct {
	agentConfig *config.AgentConfig
	args        []string
}

func (v *vipsVerifier) Validate() error {
	modelToValidate := models.VerifyVipsRequest{}
	err := ValidateCommon("vip verifier", 1, v.args, &modelToValidate)
	if err != nil {
		return err
	}

	return nil
}

func (*vipsVerifier) Command() string {
	return "vips_verifier"
}

func (v *vipsVerifier) Args() []string {
	return v.args
}

func (a *vipsVerifier) Run() (stdout, stderr string, exitCode int) {
	return vips_verifier.VerifyVips(&a.agentConfig.DryRunConfig, "", a.args...)
}
