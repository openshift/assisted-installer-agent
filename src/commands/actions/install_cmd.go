package actions

import (
	"fmt"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var podmanBaseCmd = [...]string{
	"podman", "run", "--privileged", "--pid=host", "--net=host", "--name=assisted-installer",
	"-v", "/dev:/dev:rw",
	"-v", "/opt:/opt:rw",
	"-v", "/var/log:/var/log:rw",
	"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
	"-v", "/etc/pki:/etc/pki",
	"--env=PULL_SECRET_TOKEN",
}

type install struct {
	args          []string
	installParams models.InstallCmdRequest
}

func (a *install) Validate() error {
	err := validateCommon("install", 1, a.args, &a.installParams)
	if err != nil {
		return err
	}
	return nil
}

func (a *install) CreateCmd() (string, []string) {
	installCmd := a.getFullInstallerCommand()
	return "sh", []string{"-c", installCmd}
}

func (a *install) getFullInstallerCommand() string {
	podmanCmd := podmanBaseCmd[:]

	installerCmdArgs := []string{
		"--role", string(*a.installParams.Role),
		"--infra-env-id", a.installParams.InfraEnvID.String(),
		"--cluster-id", a.installParams.ClusterID.String(),
		"--host-id", string(*a.installParams.HostID),
		"--boot-device", swag.StringValue(a.installParams.Bootdevice),
		"--url", strings.TrimSpace(swag.StringValue(a.installParams.BaseURL)),
		"--high-availability-mode", swag.StringValue(a.installParams.HighAvailabilityMode),
		"--controller-image", swag.StringValue(a.installParams.ControllerImage),
		"--agent-image", config.GlobalAgentConfig.AgentVersion,
	}

	if a.installParams.McoImage != "" {
		installerCmdArgs = append(installerCmdArgs, "--mco-image", a.installParams.McoImage)
	}

	if a.installParams.MustGatherImage != "" {
		installerCmdArgs = append(installerCmdArgs, "--must-gather-image", a.installParams.MustGatherImage)
	}

	if a.installParams.OpenshiftVersion != "" {
		installerCmdArgs = append(installerCmdArgs, "--openshift-version", a.installParams.OpenshiftVersion)
	}

	for _, diskToFormat := range a.installParams.DisksToFormat {
		installerCmdArgs = append(installerCmdArgs, "--format-disk")
		installerCmdArgs = append(installerCmdArgs, diskToFormat)
	}

	/*
		boolean flag must be used either without value (flag present means True) or in the format of <flag>=True|False.
		format <boolean flag> <value> is not supported by golang flag package and will cause the flags processing to finish
		before processing the rest of the input flags
	*/
	if swag.BoolValue(a.installParams.Insecure) {
		installerCmdArgs = append(installerCmdArgs, "--insecure")
	}

	if swag.BoolValue(a.installParams.CheckCvo) {
		installerCmdArgs = append(installerCmdArgs, "--check-cluster-version")
	}

	if a.installParams.CaCertPath != "" {
		podmanCmd = append(podmanCmd, "-v", fmt.Sprintf("%s:%s:rw", a.installParams.CaCertPath, a.installParams.CaCertPath))
		installerCmdArgs = append(installerCmdArgs, "--cacert", a.installParams.CaCertPath)
	}

	if a.installParams.InstallerArgs != "" {
		installerCmdArgs = append(installerCmdArgs, "--installer-args", a.installParams.InstallerArgs)
	}

	proxyArgs := getProxyArguments(a.installParams.Proxy)
	if len(proxyArgs) > 0 {
		installerCmdArgs = append(installerCmdArgs, proxyArgs...)
	}

	if len(a.installParams.ServiceIps) > 0 {
		installerCmdArgs = append(installerCmdArgs, "--service-ips", strings.Join(a.installParams.ServiceIps, ","))
	}

	return fmt.Sprintf("%s %s %s", shellescape.QuoteCommand(podmanCmd), swag.StringValue(a.installParams.InstallerImage),
		shellescape.QuoteCommand(installerCmdArgs))
}

func getProxyArguments(proxy *models.Proxy) []string {
	proxyArgs := make([]string, 0)
	if proxy == nil {
		return proxyArgs
	}
	httpProxy := swag.StringValue(proxy.HTTPProxy)
	httpsProxy := swag.StringValue(proxy.HTTPSProxy)
	noProxy := swag.StringValue(proxy.NoProxy)

	if httpProxy == "" && httpsProxy == "" {
		return proxyArgs
	}

	if httpProxy != "" {
		proxyArgs = append(proxyArgs, "--http-proxy", httpProxy)
	}

	if httpsProxy != "" {
		proxyArgs = append(proxyArgs, "--https-proxy", httpsProxy)
	}

	if noProxy != "" {
		proxyArgs = append(proxyArgs, "--no-proxy", strings.TrimSpace(noProxy))
	}

	return proxyArgs
}