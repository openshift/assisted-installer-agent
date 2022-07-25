package actions

import (
	"encoding/json"
	"fmt"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/spf13/afero"

	"github.com/go-openapi/runtime"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

const podman = "podman"

type ActionInterface interface {
	Validate() error
	Run() (stdout, stderr string, exitCode int)
	Command() string
	Args() []string
}

type Action struct {
	ActionInterface
}

func ValidateCommon(name string, expectedArgsLength int, args []string, modelToValidate runtime.Validatable) error {
	log.Infof("Validating %s with args %s", name, args)

	if len(args) != expectedArgsLength {
		return fmt.Errorf("%s cmd accepts %d params in args, given args %v", name, expectedArgsLength, args)
	}
	if modelToValidate != nil {
		err := json.Unmarshal([]byte(args[0]), &modelToValidate)
		if err != nil {
			log.WithError(err).Errorf("Failed to unmarshal %s: json.Unmarshal, %s", name, args[0])
			return err
		}
		err = modelToValidate.Validate(nil)
		if err != nil {
			log.WithError(err).Errorf("Failed to validate %s: data, %s", name, args[0])
			return err
		}
	}
	return nil
}

func New(agentConfig *config.AgentConfig, stepType models.StepType, args []string) (*Action, error) {
	var stepActionMap = map[models.StepType]*Action{
		models.StepTypeInventory:                  {&inventory{args: args, agentConfig: agentConfig}},
		models.StepTypeConnectivityCheck:          {&connectivityCheck{args: args, agentConfig: agentConfig}},
		models.StepTypeFreeNetworkAddresses:       {&freeAddresses{args: args, agentConfig: agentConfig}},
		models.StepTypeNtpSynchronizer:            {&ntpSynchronizer{args: args, agentConfig: agentConfig}},
		models.StepTypeInstallationDiskSpeedCheck: {&diskPerfCheck{args: args, agentConfig: agentConfig}},
		models.StepTypeAPIVipConnectivityCheck:    {&apiVipConnectivityCheck{args: args}},
		models.StepTypeTangConnectivityCheck:      {&tangConnectivityCheck{args: args}},
		models.StepTypeDhcpLeaseAllocate:          {&dhcpLeases{args: args}},
		models.StepTypeDomainResolution:           {&domainResolution{args: args}},
		models.StepTypeContainerImageAvailability: {&imageAvailability{args: args, agentConfig: agentConfig}},
		models.StepTypeStopInstallation:           {&stopInstallation{args: args}},
		models.StepTypeLogsGather:                 {&logsGather{args: args, agentConfig: agentConfig}},
		models.StepTypeInstall:                    {&install{args: args, filesystem: afero.NewOsFs(), agentConfig: agentConfig}},
		models.StepTypeUpgradeAgent:               {&upgradeAgent{args: args}},
		models.StepTypeDownloadBootArtifacts:      {&downloadBootArtifacts{args: args, agentConfig: agentConfig}},
		models.StepTypeRebootForReclaim:           {&rebootForReclaim{args: args}},
	}

	action, ok := stepActionMap[stepType]
	if !ok {
		// return error not found
		return nil, fmt.Errorf("failed to find action for step type %s", stepType)
	}
	err := action.Validate()
	return action, err
}
