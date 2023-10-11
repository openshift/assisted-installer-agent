package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/go-openapi/swag"
	"github.com/hashicorp/go-version"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/openshift/assisted-service/pkg/validations"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/thoas/go-funk"
)

var podmanBaseCmd = [...]string{
	"podman", "run", "--privileged", "--pid=host", "--net=host", "--name=assisted-installer",
	"-v", "/dev:/dev:rw",
	"-v", "/opt:/opt:rw",
	"-v", "/var/log:/var/log:rw",
	"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
	"-v", "/etc/pki:/etc/pki",
	"-v", "/tmp:/tmp",
	"--env=PULL_SECRET_TOKEN",
}

type install struct {
	args          []string
	installParams models.InstallCmdRequest
	filesystem    afero.Fs
	agentConfig   *config.AgentConfig
}

func (a *install) Validate() error {
	err := ValidateCommon("install", 1, a.args, &a.installParams)
	if err != nil {
		return err
	}

	if a.installParams.MustGatherImage != "" {
		err = validateMustGatherImages(a.installParams.MustGatherImage)
		if err != nil {
			return err
		}
	}

	if a.installParams.Proxy != nil {
		err = validateProxy(a.installParams.Proxy)
		if err != nil {
			return err
		}
	}

	if a.installParams.InstallerArgs != "" {
		var installAgs []string
		err := json.Unmarshal([]byte(a.installParams.InstallerArgs), &installAgs)
		if err != nil {
			log.WithError(err).Errorf("Failed to unmarshal installer args: json.Unmarshal, %s", a.installParams.InstallerArgs)
			return err
		}
		err = validations.ValidateInstallerArgs(installAgs)
		if err != nil {
			return err
		}
	}

	if a.installParams.OpenshiftVersion != "" {
		_, err := version.NewVersion(a.installParams.OpenshiftVersion)
		if err != nil {

			return errors.Wrapf(err, "Failed to parse OCP version %s", a.installParams.OpenshiftVersion)
		}
	}

	return a.validateDisks()
}

func (a *install) getFullInstallerCommand() string {
	podmanCmd := podmanBaseCmd[:]

	installerCmdArgs := []string{
		"--role", string(*a.installParams.Role),
		"--infra-env-id", a.installParams.InfraEnvID.String(),
		"--cluster-id", a.installParams.ClusterID.String(),
		"--host-id", string(*a.installParams.HostID),
		"--boot-device", swag.StringValue(a.installParams.BootDevice),
		"--url", a.agentConfig.TargetURL,
		"--controller-image", swag.StringValue(a.installParams.ControllerImage),
		"--agent-image", a.agentConfig.AgentVersion,
	}

	if a.installParams.HighAvailabilityMode != nil {
		installerCmdArgs = append(installerCmdArgs, "--high-availability-mode", swag.StringValue(a.installParams.HighAvailabilityMode))
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
	if a.agentConfig.InsecureConnection {
		installerCmdArgs = append(installerCmdArgs, "--insecure")
	}

	if swag.BoolValue(a.installParams.CheckCvo) {
		installerCmdArgs = append(installerCmdArgs, "--check-cluster-version")
	}

	if a.installParams.SkipInstallationDiskCleanup {
		installerCmdArgs = append(installerCmdArgs, "--skip-installation-disk-cleanup")
	}

	if a.agentConfig.CACertificatePath != "" {
		podmanCmd = append(podmanCmd, "-v", fmt.Sprintf("%s:%s:rw", a.agentConfig.CACertificatePath,
			a.agentConfig.CACertificatePath))
		installerCmdArgs = append(installerCmdArgs, "--cacert", a.agentConfig.CACertificatePath)
	}

	if a.installParams.InstallerArgs != "" {
		installerCmdArgs = append(installerCmdArgs, "--installer-args", a.installParams.InstallerArgs)
	}

	if a.installParams.EnableSkipMcoReboot {
		installerCmdArgs = append(installerCmdArgs, "--enable-skip-mco-reboot", "true")
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

func validateProxy(proxy *models.Proxy) error {
	httpProxy := swag.StringValue(proxy.HTTPProxy)
	httpsProxy := swag.StringValue(proxy.HTTPSProxy)
	noProxy := swag.StringValue(proxy.NoProxy)

	if httpProxy != "" {
		err := validations.ValidateHTTPProxyFormat(httpProxy)
		if err != nil {
			return err
		}
	}

	if httpsProxy != "" {
		err := validations.ValidateHTTPProxyFormat(httpsProxy)
		if err != nil {
			return err
		}
	}

	if noProxy != "" {
		err := validations.ValidateNoProxyFormat(noProxy)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateMustGatherImages(mustGatherImage string) error {
	var imageMap map[string]string
	err := json.Unmarshal([]byte(mustGatherImage), &imageMap)
	if err != nil {
		// must gather image can be a string and not json
		imageMap = map[string]string{"ocp": mustGatherImage}
	}
	r, errCompile := regexp.Compile(`^(([a-zA-Z0-9\-\.]+)(:[0-9]+)?\/)?[a-z0-9\._\-\/@]+[?::a-zA-Z0-9_\-.]+$`)
	if errCompile != nil {
		return errCompile
	}

	for op, image := range imageMap {
		if !r.MatchString(image) {
			return fmt.Errorf("must gather image %s validation failed %v", image, imageMap)
		}
		// TODO: adding check for supported operators
		if !funk.Contains([]string{"cnv", "lso", "ocs", "odf", "ocp"}, op) {
			return fmt.Errorf("operator name %s validation failed", op)
		}
	}
	return nil
}

func (a *install) validateDisks() error {
	disksToValidate := append(a.installParams.DisksToFormat, swag.StringValue(a.installParams.BootDevice))
	for _, disk := range disksToValidate {
		if !strings.HasPrefix(disk, "/dev/") {
			return fmt.Errorf("disk %s should start of with /dev/", disk)
		}
		if !a.pathExists(disk) {
			return fmt.Errorf("disk %s was not found on the host", disk)
		}
	}
	return nil
}

func (a *install) pathExists(path string) bool {
	if _, err := a.filesystem.Stat(path); os.IsNotExist(err) {
		return false
	} else if err != nil {
		log.WithError(err).Errorf("failed to verify path %s", path)
		return false
	}
	return true
}

func (a *install) Run() (stdout, stderr string, exitCode int) {
	return util.ExecutePrivileged(a.Command(), a.Args()...)
}

func (a *install) Command() string {
	return "sh"
}

func (a *install) Args() []string {
	return []string{"-c", a.getFullInstallerCommand()}
}
