package actions

import "github.com/openshift/assisted-installer-agent/src/util"

type stopInstallation struct {
	args []string
}

func (a *stopInstallation) Validate() error {
	err := validateCommon("stop installation", 0, a.args, nil)
	if err != nil {
		return err
	}
	return nil
}

func (a *stopInstallation) Run() (stdout, stderr string, exitCode int) {
	return util.ExecutePrivileged(a.Command(), a.Args()...)
}

func (a *stopInstallation) Command() string {
	return podman
}

func (a *stopInstallation) Args() []string {
	return []string{
		"stop", "-i", "-t", "5", "assisted-installer",
	}
}
