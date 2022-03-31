package commands

import (
	"github.com/openshift/assisted-installer-agent/src/commands/actions"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

type NextStepRunnerFactory interface {
	Create(stepType models.StepType, command string, args []string) (Runner, error)
}

// Runner is the means to allow pluggable running mechanism to agent.
// The runner factory should be initiated once, and should be passed forward in the execution floe by dependency
// injection.
type Runner interface {
	Run() (stdout, stderr string, exitCode int)
	Command() string
	Args() []string
}

func NewActionRunner(action actions.ActionInterface) Runner {
	return &actionRunner{
		action:  action,
		handler: util.Execute,
	}
}

func NewPrivilegedActionRunner(action actions.ActionInterface) Runner {
	return &actionRunner{
		action:  action,
		handler: util.ExecutePrivileged,
	}
}

type nextStepRunnerFactory struct{}

func NewNextStepRunnerFactory() NextStepRunnerFactory {
	return &nextStepRunnerFactory{}
}

func (a *nextStepRunnerFactory) Create(stepType models.StepType, command string, args []string) (Runner, error) {
	var runner Runner
	// TODO: MGMT-9451 remove command == "" after agent changes for new protocol will be pushed, added to allow pushing agent before service
	if command == "" {
		actionToRun, err := actions.New(stepType, args)
		if err != nil {
			return nil, err
		}
		runner = NewPrivilegedActionRunner(actionToRun)
	} else {
		runner = NewExecuteRunner(command, args)
	}
	return runner, nil
}

type actionRunner struct {
	action  actions.ActionInterface
	handler HandlerType
}

func (a *actionRunner) Run() (stdout, stderr string, exitCode int) {
	command, args := a.action.CreateCmd()
	return a.handler(command, args...)
}

func (a *actionRunner) Command() string {
	command, _ := a.action.CreateCmd()
	return command
}

func (a *actionRunner) Args() []string {
	_, args := a.action.CreateCmd()
	return args
}

type ExecuteRunner struct {
	command string
	args    []string
}

func (e *ExecuteRunner) Run() (stdout, stderr string, exitCode int) {
	return util.ExecutePrivileged(e.command, e.args...)
}

func (e *ExecuteRunner) Command() string {
	return e.command
}

func (e *ExecuteRunner) Args() []string {
	return e.args
}

func NewExecuteRunner(command string, args []string) Runner {
	return &ExecuteRunner{
		command: command,
		args:    args,
	}
}
