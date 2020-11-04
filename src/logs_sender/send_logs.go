package logs_sender

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/pkg/errors"

	"github.com/go-openapi/strfmt"
	"github.com/openshift/assisted-installer-agent/src/session"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/client/installer"

	log "github.com/sirupsen/logrus"
)

//go:generate mockery -name LogsSender -inpkg
type LogsSender interface {
	Execute(command string, args ...string) (stdout string, stderr string, exitCode int)
	ExecutePrivilege(command string, args ...string) (stdout string, stderr string, exitCode int)
	ExecuteOutputToFile(outputFilePath string, command string, args ...string) (stderr string, exitCode int)
	CreateFolderIfNotExist(folder string) error
	FileUploader(filePath string, clusterID strfmt.UUID, hostID strfmt.UUID,
		inventoryUrl string, pullSecretToken string, agentVersion string) error
}

type LogsSenderExecuter struct{}

func (e *LogsSenderExecuter) Execute(command string, args ...string) (stdout string, stderr string, exitCode int) {
	return util.Execute(command, args...)
}

// ExecutePrivilege execute a command in the host environment via nsenter
func (e *LogsSenderExecuter) ExecutePrivilege(command string, args ...string) (stdout string, stderr string, exitCode int) {
	commandBase := "nsenter"
	arguments := []string{"-t", "1", "-m", "-i", "-n", "--", command}
	arguments = append(arguments, args...)
	return e.Execute(commandBase, arguments...)
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

const logsDir = "/var/log"

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

func SendLogs(l LogsSender) error {
	log.Infof("Start gathering journalctl logs with tags %s and services %s",
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
		return err
	}

	for _, tag := range config.LogsSenderConfig.Tags {
		outputFile := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", tag))
		err := getJournalLogsWithFilter(l, config.LogsSenderConfig.Since, outputFile,
			[]string{fmt.Sprintf("TAG=%s", tag)})
		if err != nil {
			return err
		}
	}

	for _, service := range config.LogsSenderConfig.Services {
		outputFile := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", service))
		err := getJournalLogsWithFilter(l, config.LogsSenderConfig.Since, outputFile,
			[]string{"-u", service})
		if err != nil {
			return err
		}
	}

	if config.LogsSenderConfig.InstallerGatherlogging && config.LogsSenderConfig.IsBootstrap {
		log.Info("Running installer-gather.sh")
		// installer-gather.sh is written in such a way it always return 0.
		stdOut, stdErr, _ := l.ExecutePrivilege("/usr/local/bin/installer-gather.sh")
		log.Info(stdOut)
		if stdErr != "" {
			log.Warn(stdErr)
		}
		_, stdErr, exitCode := l.ExecutePrivilege("mv", "/root/log-bundle-.tar.gz", fmt.Sprintf("%s/installer_gather.tar.gz", logsTmpFilesDir))
		if exitCode != 0 {
			err := errors.Errorf(stdErr)
			log.WithError(err).Errorf("Failed to run installer-gather.sh command")
			return err
		}
	}

	if err := archiveFilesInFolder(l, logsTmpFilesDir, archivePath); err != nil {
		return err
	}

	return uploadLogs(l, archivePath, strfmt.UUID(config.LogsSenderConfig.ClusterID),
		strfmt.UUID(config.LogsSenderConfig.HostID),
		config.LogsSenderConfig.TargetURL, config.LogsSenderConfig.PullSecretToken, config.GlobalAgentConfig.AgentVersion)
}
