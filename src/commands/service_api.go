package commands

import (
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
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

type v2ServiceAPI struct {
	agentConfig *config.AgentConfig
}

func (v *v2ServiceAPI) RegisterHost(s *session.InventorySession) (*models.HostRegistrationResponse, error) {
	var hostID strfmt.UUID = strfmt.UUID("")
	if !v.agentConfig.DryRunEnabled {
		hostID = *scanners.ReadId(scanners.NewGHWSerialDiscovery(), agent_utils.NewDependencies(&v.agentConfig.DryRunConfig, ""))
	} else {
		hostID = strfmt.UUID(v.agentConfig.ForcedHostID)
	}

	params := &installer.V2RegisterHostParams{
		InfraEnvID:            strfmt.UUID(v.agentConfig.InfraEnvID),
		DiscoveryAgentVersion: &v.agentConfig.AgentVersion,
		NewHostParams: &models.HostCreateParams{
			HostID:                &hostID,
			DiscoveryAgentVersion: v.agentConfig.AgentVersion,
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
		HostID:                strfmt.UUID(v.agentConfig.HostID),
		InfraEnvID:            strfmt.UUID(v.agentConfig.InfraEnvID),
		DiscoveryAgentVersion: &v.agentConfig.AgentVersion,
		Timestamp:             swag.Int64(time.Now().Unix()),
	}
	result, err := s.Client().Installer.V2GetNextSteps(s.Context(), &params)
	if err != nil {
		return nil, err
	}
	return result.Payload, nil
}

func (v *v2ServiceAPI) PostStepReply(s *session.InventorySession, reply *models.StepReply) error {
	params := installer.V2PostStepReplyParams{
		HostID:                strfmt.UUID(v.agentConfig.HostID),
		InfraEnvID:            strfmt.UUID(v.agentConfig.InfraEnvID),
		DiscoveryAgentVersion: &v.agentConfig.AgentVersion,
		Reply:                 reply,
	}

	_, err := s.Client().Installer.V2PostStepReply(s.Context(), &params)
	return err
}

func newServiceAPI(agentConfig *config.AgentConfig) serviceAPI {
	return &v2ServiceAPI{
		agentConfig: agentConfig,
	}
}
