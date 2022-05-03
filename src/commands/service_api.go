package commands

import (
	"github.com/go-openapi/strfmt"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/scanners"
	"github.com/openshift/assisted-installer-agent/src/session"
	agent_utils "github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/client/installer"
	"github.com/openshift/assisted-service/models"
)

type serviceAPI interface {
	RegisterHost(s *session.InventorySession) (*models.HostRegistrationResponse, error)
	GetNextSteps(s *session.InventorySession) (*models.Steps, error)
	PostStepReply(s *session.InventorySession, reply *models.StepReply) error
}

type v2ServiceAPI struct{}

func (v *v2ServiceAPI) RegisterHost(s *session.InventorySession) (*models.HostRegistrationResponse, error) {
	var hostID strfmt.UUID = strfmt.UUID("")
	if !config.GlobalDryRunConfig.DryRunEnabled {
		hostID = *scanners.ReadId(scanners.NewGHWSerialDiscovery(), agent_utils.NewDependencies(""))
	} else {
		hostID = strfmt.UUID(config.GlobalDryRunConfig.ForcedHostID)
	}

	params := &installer.V2RegisterHostParams{
		InfraEnvID:            strfmt.UUID(config.GlobalAgentConfig.InfraEnvID),
		DiscoveryAgentVersion: &config.GlobalAgentConfig.DiscoveryAgentVersion,
		NewHostParams: &models.HostCreateParams{
			HostID:                &hostID,
			DiscoveryAgentVersion: config.GlobalAgentConfig.DiscoveryAgentVersion,
		},
	}

	result, err := s.Client().Installer.V2RegisterHost(s.Context(), params)
	if err != nil {
		return nil, err
	}
	return result.Payload, nil
}

func (v *v2ServiceAPI) GetNextSteps(s *session.InventorySession) (*models.Steps, error) {
	params := installer.V2GetNextStepsParams{
		HostID:                strfmt.UUID(config.GlobalAgentConfig.HostID),
		InfraEnvID:            strfmt.UUID(config.GlobalAgentConfig.InfraEnvID),
		DiscoveryAgentVersion: &config.GlobalAgentConfig.DiscoveryAgentVersion,
	}
	result, err := s.Client().Installer.V2GetNextSteps(s.Context(), &params)
	if err != nil {
		return nil, err
	}
	return result.Payload, nil
}

func (v *v2ServiceAPI) PostStepReply(s *session.InventorySession, reply *models.StepReply) error {
	params := installer.V2PostStepReplyParams{
		HostID:                strfmt.UUID(config.GlobalAgentConfig.HostID),
		InfraEnvID:            strfmt.UUID(config.GlobalAgentConfig.InfraEnvID),
		DiscoveryAgentVersion: &config.GlobalAgentConfig.DiscoveryAgentVersion,
		Reply:                 reply,
	}

	_, err := s.Client().Installer.V2PostStepReply(s.Context(), &params)
	return err
}

func newServiceAPI() serviceAPI {
	return &v2ServiceAPI{}
}
