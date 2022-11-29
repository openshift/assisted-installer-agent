package disk_speed_check

import (
	"github.com/openshift/assisted-installer-agent/src/util"
)

//go:generate mockery --name IDependencies --inpackage
type IDependencies interface {
	Execute(command string, args ...string) (stdout string, stderr string, exitCode int)
}

type Dependencies struct{}

func (d *Dependencies) Execute(command string, args ...string) (stdout string, stderr string, exitCode int) {
	return util.Execute(command, args...)
}

func NewDependencies() IDependencies {
	return &Dependencies{}
}
