package config

// ConnectivityConfig defines minimal configuration for connecting to assisted service
type ConnectivityConfig struct {
	TargetURL             string
	ClusterID             string
	InfraEnvID            string
	AgentVersion          string
	PullSecretToken       string
	InsecureConnection    bool
	CACertificatePath     string
	DiscoveryAgentVersion string
}
