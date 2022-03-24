package actions

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

func (a *stopInstallation) CreateCmd() (string, []string) {
	podmanRunCmd := []string{
		"stop", "-i", "-t", "5", "assisted-installer",
	}
	return podman, podmanRunCmd
}
