package commands

import (
	"github.com/openshift/assisted-installer-agent/src/commands/actions"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

type NextStepRunnerFactory interface {
	Create(agentConfig *config.AgentConfig, command string, args []string) (Runner, error)
}

type ToolRunnerFactory interface {
	Create(agentConfig *config.AgentConfig, stepType models.StepType, command string, args []string) (Runner, error)
}

// Runner is the means to allow pluggable running mechanism to agent.
// The runner factory should be initiated once, and should be passed forward in the execution flow by dependency
// injection.
type Runner interface {
	Run() (stdout, stderr string, exitCode int)
	Command() string
	Args() []string
}

type toolRunnerFactory struct{}

func NewToolRunnerFactory() ToolRunnerFactory {
	return &toolRunnerFactory{}
}

func (a *toolRunnerFactory) Create(agentConfig *config.AgentConfig, stepType models.StepType, command string, args []string) (Runner, error) {
	// TODO: MGMT-9451 remove command == "" after agent changes for new protocol will be pushed, added to allow pushing agent before service
	if command == "" {
		actionToRun, err := actions.New(agentConfig, stepType, args)
		if err != nil {
			return nil, err
		}
		return actionToRun, nil
	}
	return NewPrivilegedExecuteRunner(command, args), nil
}

type executeRunner struct {
	command string
	args    []string
	handler HandlerType
}

func (e *executeRunner) Run() (stdout, stderr string, exitCode int) {
	return e.handler(e.command, e.args...)
}

func (e *executeRunner) Command() string {
	return e.command
}

func (e *executeRunner) Args() []string {
	return e.args
}

func NewPrivilegedExecuteRunner(command string, args []string) Runner {
	return &executeRunner{
		command: command,
		args:    args,
		handler: util.ExecutePrivileged,
	}
}

func NewExecuteRunner(command string, args []string) Runner {
	return &executeRunner{
		command: command,
		args:    args,
		handler: util.Execute,
	}
}
