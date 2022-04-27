package commands

import (
	"github.com/openshift/assisted-installer-agent/src/commands/actions"
	"github.com/openshift/assisted-installer-agent/src/config"
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
	actionToRun, err := actions.New(agentConfig, stepType, args)
	if err != nil {
		return nil, err
	}
	return actionToRun, nil
}
