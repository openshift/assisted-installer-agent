package commands

import (
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func StartStepRunner(command string, args ...string) error {

	log.Infof("Running next step runner. Command: %s, Args: %q", command, args)
	_, stderr, exitCode := util.Execute(command, args...)
	if exitCode != 0 {
		return errors.Errorf("next step runner command exited with non-zero exit code %d: %s", exitCode, stderr)
	}
	return nil
}
