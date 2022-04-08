package logs_sender

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/pkg/errors"

	"github.com/go-openapi/strfmt"
	"github.com/openshift/assisted-installer-agent/src/session"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/client"
	"github.com/openshift/assisted-service/client/installer"
	"github.com/openshift/assisted-service/models"

	log "github.com/sirupsen/logrus"
)

const (
	logsDir                      = "/var/log"
	installerGatherBin           = "/usr/local/bin/installer-gather.sh"
	ovsGatherBin                 = "/usr/local/bin/ovs-installer-gather.sh"
	installerGatherArchivePreifx = "/root/log-bundle-"
	lsblk                        = "/usr/bin/lsblk"
	findmnt                      = "/usr/bin/findmnt"
	ls                           = "/bin/ls"
	pvdisplay                    = "/usr/sbin/pvdisplay"
	vgdisplay                    = "/usr/sbin/vgdisplay"
	lvdisplay                    = "/usr/sbin/lvdisplay"
)

//go:generate mockery -name LogsSender -inpkg
type LogsSender interface {
	Execute(command string, args ...string) (stdout string, stderr string, exitCode int)
	ExecutePrivileged(command string, args ...string) (stdout string, stderr string, exitCode int)
	ExecuteOutputToFile(outputFilePath string, command string, args ...string) (stderr string, exitCode int)
	CreateFolderIfNotExist(folder string) error
	FileUploader(filePath string, clusterID strfmt.UUID, hostID strfmt.UUID, infraEnvID strfmt.UUID,
		inventoryUrl string, pullSecretToken string) error
	LogProgressReport(infraEnvID strfmt.UUID, hostID strfmt.UUID, inventoryUrl string, pullSecretToken string, progress models.LogsState) error
	GatherInstallerLogs(targetDir string) error
	GatherErrorLogs(targetDir string) error
}

type LogsSenderExecuter struct {
	client        *client.AssistedInstall
	ctx           context.Context
	agentVersion  string
	loggingConfig *config.LogsSenderConfig
}

func NewLogsSenderExecuter(loggingConfig *config.LogsSenderConfig, inventoryUrl string, pullSecretToken string, agentVersion string) *LogsSenderExecuter {
	client, ctx := getClient(loggingConfig, inventoryUrl, pullSecretToken)
	return &LogsSenderExecuter{
		client:        client,
		ctx:           ctx,
		agentVersion:  agentVersion,
		loggingConfig: loggingConfig,
	}
}

func (e *LogsSenderExecuter) Execute(command string, args ...string) (stdout string, stderr string, exitCode int) {
	return util.Execute(command, args...)
}

// ExecutePrivileged execute a command in the host environment via nsenter
func (e *LogsSenderExecuter) ExecutePrivileged(command string, args ...string) (stdout string, stderr string, exitCode int) {
	return util.ExecutePrivileged(command, args...)
}

func (e *LogsSenderExecuter) ExecuteOutputToFile(outputFilePath string, command string, args ...string) (stderr string, exitCode int) {
	return util.ExecuteOutputToFile(outputFilePath, command, args...)
}

func (e *LogsSenderExecuter) CreateFolderIfNotExist(folder string) error {
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		return os.MkdirAll(folder, 0755)
	}
	return nil
}

func getClient(loggingConfig *config.LogsSenderConfig, inventoryUrl string, pullSecretToken string) (*client.AssistedInstall, context.Context) {
	invSession, err := session.New(&loggingConfig.AgentConfig, inventoryUrl, pullSecretToken)
	if err != nil {
		log.Fatalf("Failed to initialize connection: %e", err)
	}
	return invSession.Client(), invSession.Context()
}

func (e *LogsSenderExecuter) FileUploader(filePath string, clusterID strfmt.UUID, hostID strfmt.UUID, infraEnvID strfmt.UUID,
	inventoryUrl string, pullSecretToken string) error {

	uploadFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer uploadFile.Close()

	params := installer.V2UploadLogsParams{
		Upfile:     uploadFile,
		ClusterID:  clusterID,
		HostID:     &hostID,
		InfraEnvID: &infraEnvID,
		LogsType:   string(models.LogsTypeHost),
	}
	_, err = e.client.Installer.V2UploadLogs(e.ctx, &params)
	return err
}

func (e *LogsSenderExecuter) LogProgressReport(infraEnvID strfmt.UUID, hostID strfmt.UUID, inventoryUrl string, pullSecretToken string, progress models.LogsState) error {
	params := installer.V2UpdateHostLogsProgressParams{
		InfraEnvID: infraEnvID,
		HostID:     hostID,
		LogsProgressParams: &models.LogsProgressParams{
			LogsState: &progress,
		},
	}

	_, err := e.client.Installer.V2UpdateHostLogsProgress(e.ctx, &params)
	return err
}

func (e *LogsSenderExecuter) GatherInstallerLogs(targetDir string) error {
	var result, err error

	gatherID := time.Now().Format("20060102150405")
	mastersIPs := strings.Split(e.loggingConfig.MastersIPs, ",")
	//Create ovs logs where installer-gather expects to find its input.
	//Then, installer-gather.sh runa as overlay and finally bundles
	//all logs together. Unlike installer-gather.sh, ovs-installer-gather.sh
	//runs locally in the container rather on the host
	ovsGatherArgs := append([]string{"--id", gatherID}, mastersIPs...)
	log.Infof("Running %s %v", ovsGatherBin, ovsGatherArgs)
	stdOut, stdErr, exitCode := e.Execute(ovsGatherBin, ovsGatherArgs...)
	for _, so := range strings.Split(stdOut, "\n") {
		log.Infof("ovs-gather log: %s", so)
	}
	if stdErr != "" || exitCode != 0 {
		err = errors.New(stdErr)
		log.WithError(err).Warnf("Failed to run %s %v", ovsGatherBin, ovsGatherArgs)
		result = multierror.Append(result, err)
	}

	installerGatherArgs := append([]string{"--id", gatherID}, mastersIPs...)
	log.Infof("Running %s %v", installerGatherBin, installerGatherArgs)
	// installer-gather.sh is written in such a way it always return 0.
	stdOut, stdErr, exitCode = e.ExecutePrivileged(installerGatherBin, installerGatherArgs...)
	for _, so := range strings.Split(stdOut, "\n") {
		log.Infof("installer-gather log: %s", so)
	}
	if stdErr != "" || exitCode != 0 {
		err = errors.New(stdErr)
		log.WithError(errors.New(stdErr)).Warnf("Failed to run %s %v", installerGatherBin, installerGatherArgs)
		result = multierror.Append(result, err)
	}

	_, stdErr, exitCode = e.ExecutePrivileged("mv", fmt.Sprintf("%s%s.tar.gz", installerGatherArchivePreifx, gatherID), targetDir)
	if exitCode != 0 {
		err = errors.New(stdErr)
		log.WithError(err).Errorf("Failed to: mv %s %s", fmt.Sprintf("%s%s.tar.gz", installerGatherArchivePreifx, gatherID), targetDir)
		result = multierror.Append(result, err)
	}
	return result
}

func (e *LogsSenderExecuter) GatherErrorLogs(targetDir string) error {
	var result error

	// Write the entire output of dmesg
	outputFile := path.Join(targetDir, "dmesg.logs")
	if err := getDmesgLogs(e, outputFile); err != nil {
		result = multierror.Append(result, err)
	}

	// Write coredump files
	if err := getCoreDumps(e, targetDir); err != nil {
		result = multierror.Append(result, err)
	}

	// Write the entire output of journal
	outputFile = path.Join(targetDir, "journal.logs")
	if err := getJournalLogs(e, e.loggingConfig.Since, outputFile, []string{}); err != nil {
		result = multierror.Append(result, err)
	}

	return result
}

func getMountLogs(l LogsSender, outputFilePath string) error {
	var result error

	logfile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer logfile.Close()

	result = util.LogPrivilegedCommandOutput(logfile, result, "List block devices", lsblk, "-o", "NAME,MAJ:MIN,SIZE,TYPE,FSTYPE,KNAME,MODEL,UUID,WWN,HCTL,VENDOR,STATE,TRAN,PKNAME")
	result = util.LogPrivilegedCommandOutput(logfile, result, "List mounts", findmnt, "--df")
	result = logDisksByCategory("id", logfile, result)
	result = logDisksByCategory("path", logfile, result)
	result = util.LogPrivilegedCommandOutput(logfile, result, "Running pvdisplay", pvdisplay, "-v")
	result = util.LogPrivilegedCommandOutput(logfile, result, "Running vgdisplay", vgdisplay, "-v")
	result = util.LogPrivilegedCommandOutput(logfile, result, "Running lvdisplay", lvdisplay, "-v")

	return result
}

func logDisksByCategory(category string, logfile *os.File, result error) error {
	path := fmt.Sprintf("/dev/disk/by-%s", category)
	description := fmt.Sprintf("Disk mapping by %s", category)
	return util.LogPrivilegedCommandOutput(logfile, result, description, ls, "-l", path)
}

func getDmesgLogs(l LogsSender, outputFilePath string) error {
	log.Infof("Running dmesg")
	stderr, exitCode := l.ExecuteOutputToFile(outputFilePath, "dmesg", "-T")
	if exitCode != 0 {
		err := errors.Errorf(stderr)
		log.WithError(err).Errorf("Failed to run dmesg command")
		return err
	}
	return nil
}

func getCoreDumps(l LogsSender, targetDir string) error {
	log.Infof("Get coredump files")
	stdout, stderr, exitCode := l.ExecutePrivileged("coredumpctl", "list", "--no-legend")
	if exitCode != 0 {
		log.Infof("Couldn't fetch coredump list: %s", stderr)
		return nil
	}

	dumps := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	for _, dump := range dumps {
		fields := strings.Fields(dump)
		pid := fields[4]
		exe := filepath.Base(fields[9])
		outputFile := path.Join(targetDir, fmt.Sprintf("coredump_exe_%s_pid_%s", exe, pid))
		_, stderr, exitCode := l.ExecutePrivileged("coredumpctl", "dump", pid, "--output", outputFile)
		if exitCode != 0 {
			err := errors.Errorf(stderr)
			log.WithError(err).Errorf("Failed to read coredump for PID: %s", pid)
			return err
		}
	}

	return nil
}

func getJournalLogs(l LogsSender, since string, outputFilePath string, journalFilterParams []string) error {
	log.Infof("Running journalctl %s", journalFilterParams)
	args := []string{"-D", "/var/log/journal/", "--since", since, "--all"}
	args = append(args, journalFilterParams...)
	stderr, exitCode := l.ExecuteOutputToFile(outputFilePath, "journalctl", args...)
	if exitCode != 0 {
		err := errors.Errorf(stderr)
		log.WithError(err).Errorf("Failed to run journalctl command")
		return err
	}
	return nil
}

func archiveFilesInFolder(l LogsSender, inputPath string, outputFile string) error {
	log.Infof("Archiving %s and creating %s", inputPath, outputFile)
	args := []string{"-czvf", outputFile, "-C", filepath.Dir(inputPath), filepath.Base(inputPath)}

	_, err, execCode := l.Execute("tar", args...)

	if execCode != 0 {
		log.WithError(errors.Errorf(err)).Errorf("Failed to run to archive %s.", inputPath)
		return fmt.Errorf(err)
	}
	return nil
}

func uploadLogs(l LogsSender, filepath string, clusterID strfmt.UUID, hostId strfmt.UUID, infraEnvID strfmt.UUID,
	inventoryUrl string, pullSecretToken string) error {

	err := l.FileUploader(filepath, clusterID, hostId, infraEnvID, inventoryUrl, pullSecretToken)
	if err != nil {
		log.WithError(err).Errorf("Failed to upload file %s to assisted-service", filepath)
		return err
	}
	return nil
}

func SendLogs(loggingConfig *config.LogsSenderConfig, l LogsSender) (error, string) {
	var result error

	if lerr := l.LogProgressReport(strfmt.UUID(loggingConfig.InfraEnvID),
		strfmt.UUID(loggingConfig.HostID), loggingConfig.TargetURL,
		loggingConfig.PullSecretToken, models.LogsStateRequested); lerr != nil {
		log.WithError(lerr).Error("failed to send log progress requested to service")
	}

	log.Infof("Start gathering journalctl logs with tags %s, services %s and installer-gather",
		loggingConfig.Tags, loggingConfig.Services)
	archivePath := fmt.Sprintf("%s/logs.tar.gz", logsDir)
	logsTmpFilesDir := path.Join(logsDir, fmt.Sprintf("logs_host_%s", loggingConfig.HostID))

	defer func() {
		if loggingConfig.CleanWhenDone {
			_ = os.RemoveAll(logsTmpFilesDir)
			_ = os.Remove(archivePath)
		}
	}()
	if err := l.CreateFolderIfNotExist(logsTmpFilesDir); err != nil {
		log.WithError(err).Errorf("Failed to create directory %s", logsTmpFilesDir)
		return err, ""
	}

	if loggingConfig.InstallerGatherlogging {
		if loggingConfig.IsBootstrap {
			if err := l.GatherInstallerLogs(logsTmpFilesDir); err != nil {
				log.WithError(err).Error("Failed to gather installer logs")
				result = multierror.Append(result, err)
			}
		}

		if err := l.GatherErrorLogs(logsTmpFilesDir); err != nil {
			log.WithError(err).Error("Failed to gather coredumps and dmesg (ignoring for getting other logs)")
			result = multierror.Append(result, err)
		}
	}

	outputFile := path.Join(logsTmpFilesDir, "mount.logs")
	if err := getMountLogs(l, outputFile); err != nil {
		result = multierror.Append(result, err)
	}

	for _, tag := range loggingConfig.Tags {
		outputFile := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", tag))
		if err := getJournalLogs(l, loggingConfig.Since, outputFile,
			[]string{fmt.Sprintf("TAG=%s", tag)}); err != nil {
			result = multierror.Append(result, err)
		}
	}

	for _, service := range loggingConfig.Services {
		outputFile := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", service))
		if err := getJournalLogs(l, loggingConfig.Since, outputFile,
			[]string{"-u", service}); err != nil {
			result = multierror.Append(result, err)
		}
	}

	var report = ""
	if result != nil {
		report = result.Error()
	}

	if err := archiveFilesInFolder(l, logsTmpFilesDir, archivePath); err != nil {
		return err, report
	}

	err := uploadLogs(l, archivePath, strfmt.UUID(loggingConfig.ClusterID),
		strfmt.UUID(loggingConfig.HostID), strfmt.UUID(loggingConfig.InfraEnvID),
		loggingConfig.TargetURL, loggingConfig.PullSecretToken)

	if lerr := l.LogProgressReport(strfmt.UUID(loggingConfig.InfraEnvID),
		strfmt.UUID(loggingConfig.HostID), loggingConfig.TargetURL,
		loggingConfig.PullSecretToken, models.LogsStateCompleted); lerr != nil {
		log.WithError(lerr).Error("failed to send log progress completed to service")
	}

	return err, report
}
