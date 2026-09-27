package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/djherbis/times"
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

const (
	templatePull              = "podman pull %s"
	templateTimeout           = "timeout %s %s"
	templateGetImage          = "podman images --quiet %s"
	failedToPullImageExitCode = 2
	defaultImagePullRetries   = 3
	defaultImagePullTimeout   = 600
	// nmConnectionsDir uses /proc/1/root to access the host's filesystem from
	// within the next-step-runner container, which runs with --pid=host but does
	// not mount /etc/NetworkManager/system-connections directly.
	nmConnectionsDir = "/proc/1/root/etc/NetworkManager/system-connections"
	// agentTUILogFile and agentTUILogDir are accessible at their normal paths
	// because the next-step-runner container mounts /var/log from the host.
	agentTUILogFile = "/var/log/agent/agent-tui.log"
	agentTUILogDir  = "/var/log/agent"
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
	birthTimeFn   func(string) (time.Time, bool)
}

// defaultBirthTimeFn returns the birth time of the file at the given path using
// the statx syscall (kernel >= 4.11). Returns false if birth time is unavailable.
func defaultBirthTimeFn(path string) (time.Time, bool) {
	ts, err := times.Stat(path)
	if err != nil || !ts.HasBirthTime() {
		return time.Time{}, false
	}
	return ts.BirthTime(), true
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
		err = json.Unmarshal([]byte(a.installParams.InstallerArgs), &installAgs)
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
		_, err = version.NewVersion(a.installParams.OpenshiftVersion)
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

	if a.installParams.ControlPlaneCount != 0 {
		installerCmdArgs = append(installerCmdArgs, "--control-plane-count", strconv.Itoa(int(a.installParams.ControlPlaneCount)))
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

	if installerArgs := a.buildInstallerArgs(); len(installerArgs) > 0 {
		argsJSON, err := json.Marshal(installerArgs)
		if err != nil {
			log.WithError(err).Warn("Failed to marshal installer args, skipping")
		} else {
			installerCmdArgs = append(installerCmdArgs, "--installer-args", string(argsJSON))
		}
	}

	if a.installParams.EnableSkipMcoReboot {
		installerCmdArgs = append(installerCmdArgs, "--enable-skip-mco-reboot")
	}

	if a.installParams.NotifyNumReboots {
		installerCmdArgs = append(installerCmdArgs, "--notify-num-reboots")
	}

	proxyArgs := getProxyArguments(a.installParams.Proxy)
	if len(proxyArgs) > 0 {
		installerCmdArgs = append(installerCmdArgs, proxyArgs...)
	}

	if len(a.installParams.ServiceIps) > 0 {
		installerCmdArgs = append(installerCmdArgs, "--service-ips", strings.Join(a.installParams.ServiceIps, ","))
	}

	if len(a.installParams.CoreosImage) > 0 {
		installerCmdArgs = append(installerCmdArgs, "--coreos-image", a.installParams.CoreosImage)
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

// agentTUIStartTime returns the time the agent-tui started on this node.
// It checks for the existence of the agent-tui log file to confirm the TUI
// actually ran, then returns the birth time of the agent log directory.
//
// The log directory is created by agent-interactive-console.service's ExecStartPre
// (mkdir -p /var/log/agent) just before the TUI launches. On an agent ISO boot the
// directory does not exist beforehand, so its birth time is set exactly once at
// service startup — making it a reliable proxy for TUI start time. Birth time is
// used rather than mtime because mtime advances whenever a file is created inside
// the directory (e.g. when agent-tui.log is created).
//
// Returns a zero time if the log file does not exist (not an ABI workflow or the
// TUI did not produce output) or if birth time is unavailable on this system.
func (a *install) agentTUIStartTime() time.Time {
	if _, err := a.filesystem.Stat(agentTUILogFile); err != nil {
		return time.Time{}
	}
	birthTimeFn := a.birthTimeFn
	if birthTimeFn == nil {
		birthTimeFn = defaultBirthTimeFn
	}
	btime, ok := birthTimeFn(agentTUILogDir)
	if !ok {
		return time.Time{}
	}
	return btime
}

// hasManualNetworkConfig returns true if NetworkManager keyfiles were created
// during the agent-tui session on this node.
//
// The agent-tui gives users access to nmtui to configure networking. Any keyfile
// the user creates via nmtui will appear in the NM connections directory AFTER the
// agent-tui started. Auto-generated files (nm-initrd-generator, NMState configs via
// pre-network-manager-config.sh) are created BEFORE the agent-tui starts, so they
// are excluded by comparing their mtime to the TUI start time.
//
// If the agent-tui log file does not exist the TUI did not run on this node, so
// we skip the check entirely. This limits the logic to ABI TUI scenarios and avoids
// false positives in non-ABI workflows.
//
// Note: when NMState static configs are provided via agent-config.yaml, assisted-service
// already adds --copy-network via StaticNetworkConfig, so those keyfiles do not need
// to be detected here.
func (a *install) hasManualNetworkConfig() bool {
	tuiStart := a.agentTUIStartTime()
	if tuiStart.IsZero() {
		log.Info("Agent TUI log file not found, skipping manual network config detection")
		return false
	}

	entries, err := afero.ReadDir(a.filesystem, nmConnectionsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warnf("Could not read %s: %v", nmConnectionsDir, err)
		}
		return false
	}
	log.Infof("Checking %s for manually-created keyfiles (agent-tui started at %v)", nmConnectionsDir, tuiStart)
	found := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := filepath.Join(nmConnectionsDir, entry.Name())

		// Only consider files created at or after the agent-tui started. All auto-generated
		// keyfiles (nm-initrd-generator, NMState via pre-network-manager-config.sh)
		// are written before the TUI launches and will have an earlier mtime. We use
		// >= rather than > to handle filesystems with second-level mtime precision
		// where the user's keyfile and the TUI start may share the same timestamp.
		info, err := a.filesystem.Stat(filePath)
		if err != nil {
			continue
		}
		log.Infof("Found NM keyfile: %s (mtime %v)", filePath, info.ModTime())
		if info.ModTime().Before(tuiStart) {
			log.Infof("Skipping NM keyfile created before agent-tui started (mtime %v < TUI start %v): %s",
				info.ModTime(), tuiStart, filePath)
			continue
		}

		content, err := afero.ReadFile(a.filesystem, filePath)
		if err != nil {
			continue
		}
		if strings.Contains(string(content), "[connection]") {
			log.Infof("Found manually-created NetworkManager keyfile: %s (mtime %v)", filePath, info.ModTime())
			found = true
			break
		}
	}
	if !found {
		log.Infof("No manually-created NetworkManager keyfiles found in %s", nmConnectionsDir)
	}
	return found
}

// buildInstallerArgs returns the installer args to pass to coreos-installer,
// combining any args already set via the assisted-service API with --copy-network
// if manually-created NetworkManager keyfiles are detected on this host.
func (a *install) buildInstallerArgs() []string {
	var args []string
	if a.installParams.InstallerArgs != "" {
		if err := json.Unmarshal([]byte(a.installParams.InstallerArgs), &args); err != nil {
			log.WithError(err).Warnf("Failed to unmarshal installer args: %s", a.installParams.InstallerArgs)
		}
	}
	if a.hasManualNetworkConfig() && !funk.Contains(args, "--copy-network") {
		log.Infof("Manually-created NetworkManager keyfiles detected in %s, adding --copy-network",
			nmConnectionsDir)
		args = append(args, "--copy-network")
	}
	return args
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
	if err := downloadInstallerImage(*a.installParams.InstallerImage); err != nil {
		return "", err.Error(), failedToPullImageExitCode
	}

	return util.ExecutePrivileged(a.Command(), a.Args()...)
}

func (a *install) Command() string {
	return "sh"
}

func (a *install) Args() []string {
	return []string{"-c", a.getFullInstallerCommand()}
}

func downloadInstallerImage(image string) error {
	if !isImageAvailable(image) {
		if err := pullImageWithRetry(defaultImagePullTimeout, image, defaultImagePullRetries); err != nil {
			return err
		}
	}
	return nil
}

func isImageAvailable(image string) bool {
	cmd := fmt.Sprintf(templateGetImage, image)
	args := strings.Split(cmd, " ")
	stdout, _, exitCode := util.ExecutePrivileged(args[0], args[1:]...)
	return exitCode == 0 && stdout != ""
}

func pullImageWithRetry(pullTimeoutSeconds int64, image string, retry int) error {
	var err error
	for attempts := 0; attempts < retry; attempts++ {
		if err = pullImage(pullTimeoutSeconds, image); err == nil {
			break
		}
	}
	return err
}

func pullImage(pullTimeoutSeconds int64, image string) error {
	cmd := fmt.Sprintf(templatePull, image)
	cmd = fmt.Sprintf(templateTimeout, strconv.FormatInt(pullTimeoutSeconds, 10), cmd)
	args := strings.Split(cmd, " ")
	stdout, stderr, exitCode := util.ExecutePrivileged(args[0], args[1:]...)

	switch exitCode {
	case 0:
		return nil
	case util.TimeoutExitCode:
		return errors.Errorf("pulling the installer image %s timed out after %d seconds", image, pullTimeoutSeconds)
	default:
		return errors.Errorf("pulling the installer image %s exited with non-zero exit code %d: %s\n %s", image, exitCode, stdout, stderr)
	}
}
