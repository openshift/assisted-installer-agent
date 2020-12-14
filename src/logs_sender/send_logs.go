package logs_sender

import (
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
	"github.com/openshift/assisted-service/client/installer"

	log "github.com/sirupsen/logrus"
)

const (
	logsDir                      = "/var/log"
	installerGatherBin           = "/usr/local/bin/installer-gather.sh"
	installerGatherArchivePreifx = "/root/log-bundle-"
	findmnt                      = "/usr/bin/findmnt"
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
	FileUploader(filePath string, clusterID strfmt.UUID, hostID strfmt.UUID,
		inventoryUrl string, pullSecretToken string, agentVersion string) error
	GatherInstallerLogs(targetDir string) error
	GatherErrorLogs(targetDir string) error
}

type LogsSenderExecuter struct{}

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

func (e *LogsSenderExecuter) FileUploader(filePath string, clusterID strfmt.UUID, hostID strfmt.UUID,
	inventoryUrl string, pullSecretToken string, agentVersion string) error {

	uploadFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer uploadFile.Close()

	invSession, err := session.New(inventoryUrl, pullSecretToken)
	if err != nil {
		log.Fatalf("Failed to initialize connection: %e", err)
	}

	params := installer.UploadHostLogsParams{
		Upfile:                uploadFile,
		ClusterID:             clusterID,
		DiscoveryAgentVersion: &agentVersion,
		HostID:                hostID,
	}
	_, err = invSession.Client().Installer.UploadHostLogs(invSession.Context(), &params)

	return err
}

func (e *LogsSenderExecuter) GatherInstallerLogs(targetDir string) error {
	gatherID := time.Now().Format("20060102150405")
	mastersIPs := strings.Split(config.LogsSenderConfig.MastersIPs, ",")
	installerGatherArgs := append([]string{"--id", gatherID}, mastersIPs...)
	log.Infof("Running %s %v", installerGatherBin, installerGatherArgs)
	// installer-gather.sh is written in such a way it always return 0.
	stdOut, stdErr, exitCode := e.ExecutePrivileged(installerGatherBin, installerGatherArgs...)
	for _, so := range strings.Split(stdOut, "\n") {
		log.Infof("installer-gather log: %s", so)
	}
	if stdErr != "" || exitCode != 0 {
		log.WithError(errors.New(stdErr)).Warnf("Failed to run %s %v", installerGatherBin, installerGatherArgs)
	}
	_, stdErr, exitCode = e.ExecutePrivileged("mv", fmt.Sprintf("%s%s.tar.gz", installerGatherArchivePreifx, gatherID), targetDir)
	if exitCode != 0 {
		err := errors.New(stdErr)
		log.WithError(err).Errorf("Failed to: mv %s %s", fmt.Sprintf("%s%s.tar.gz", installerGatherArchivePreifx, gatherID), targetDir)
		return err
	}
	return nil
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

	return result
}

func getMountLogs(l LogsSender, outputFilePath string) error {
	var result error

	logfile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer logfile.Close()
	
	log.Infof("Running findmnt")
	if err = util.ExecutePrivilegedToFile(logfile, findmnt, "--df"); err != nil {
		result = multierror.Append(result, err)
	}
	
	log.Infof("Running pvdisplay")
	if err = util.ExecutePrivilegedToFile(logfile, pvdisplay, "-v"); err != nil {
		result = multierror.Append(result, err)
	}

	log.Infof("Running vgdisplay")
	if err = util.ExecutePrivilegedToFile(logfile, vgdisplay, "-v"); err != nil {
		result = multierror.Append(result, err)
	}

	log.Infof("Running lvdisplay")
	if err = util.ExecutePrivilegedToFile(logfile, lvdisplay, "-v"); err != nil {
		result = multierror.Append(result, err)
	}

	return result
}

func getDmesgLogs(l LogsSender, outputFilePath string) error {
	log.Infof("Running dmesg")
	stderr, exitCode := l.ExecuteOutputToFile(outputFilePath, "dmesg")
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

func getJournalLogsWithFilter(l LogsSender, since string, outputFilePath string, journalFilterParams []string) error {
	log.Infof("Running journalctl with filters %s", journalFilterParams)
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

func uploadLogs(l LogsSender, filepath string, clusterID strfmt.UUID, hostId strfmt.UUID,
	inventoryUrl string, pullSecretToken string, agentVersion string) error {

	err := l.FileUploader(filepath, clusterID, hostId, inventoryUrl, pullSecretToken, agentVersion)
	if err != nil {
		log.WithError(err).Errorf("Failed to upload file %s to assisted-service", filepath)
		return err
	}
	return nil
}

func SendLogs(l LogsSender) (error, string) {
	var result error
	log.Infof("Start gathering journalctl logs with tags %s, services %s and installer-gather",
		config.LogsSenderConfig.Tags, config.LogsSenderConfig.Services)	
	archivePath := fmt.Sprintf("%s/logs.tar.gz", logsDir)
	logsTmpFilesDir := path.Join(logsDir, fmt.Sprintf("logs_host_%s", config.LogsSenderConfig.HostID))

	defer func() {
		if config.LogsSenderConfig.CleanWhenDone {
			_ = os.RemoveAll(logsTmpFilesDir)
			_ = os.Remove(archivePath)
		}
	}()
	if err := l.CreateFolderIfNotExist(logsTmpFilesDir); err != nil {
		log.WithError(err).Errorf("Failed to create directory %s", logsTmpFilesDir)
		return err, ""
	}

	if config.LogsSenderConfig.InstallerGatherlogging {
		if config.LogsSenderConfig.IsBootstrap {
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

	for _, tag := range config.LogsSenderConfig.Tags {
		outputFile := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", tag))
		if err := getJournalLogsWithFilter(l, config.LogsSenderConfig.Since, outputFile,
			[]string{fmt.Sprintf("TAG=%s", tag)}); err != nil {
				result = multierror.Append(result, err)
			}
	}

	for _, service := range config.LogsSenderConfig.Services {
		outputFile := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", service))
		if err := getJournalLogsWithFilter(l, config.LogsSenderConfig.Since, outputFile,
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

	return uploadLogs(l, archivePath, strfmt.UUID(config.LogsSenderConfig.ClusterID),
		strfmt.UUID(config.LogsSenderConfig.HostID), config.LogsSenderConfig.TargetURL,
		config.LogsSenderConfig.PullSecretToken, config.GlobalAgentConfig.AgentVersion), report
}
