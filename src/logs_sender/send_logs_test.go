package logs_sender

import (
	"fmt"
	"path"
	"path/filepath"
	"testing"

	strfmt "github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/pkg/errors"
)

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "logs sender unit tests")
}

var _ = Describe("logs sender", func() {

	var logsSenderMock *MockLogsSender
	var archivePath string
	var logsTmpFilesDir string

	BeforeEach(func() {
		logsSenderMock = &MockLogsSender{}
		archivePath = fmt.Sprintf("%s/logs.tar.gz", logsDir)
		config.LogsSenderConfig.Tags = []string{"agent", "installer"}
		config.LogsSenderConfig.Services = []string{"test", "test1"}
		config.LogsSenderConfig.Since = "5 seconds ago"
		config.LogsSenderConfig.TargetURL = "http://test.com"
		config.LogsSenderConfig.PullSecretToken = "test"
		config.LogsSenderConfig.ClusterID = uuid.New().String()
		config.LogsSenderConfig.HostID = uuid.New().String()
		logsTmpFilesDir = path.Join(logsDir, fmt.Sprintf("logs_host_%s", config.LogsSenderConfig.HostID))

	})

	folderSuccess := func() {
		logsSenderMock.On("CreateFolderIfNotExist", logsTmpFilesDir).
			Return(nil)
	}

	executeOutputToFileSuccess := func() {
		for _, tag := range config.LogsSenderConfig.Tags {
			outputPath := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", tag))
			logsSenderMock.On("ExecuteOutputToFile", outputPath, "journalctl", "-D", "/var/log/journal/",
				"--since", config.LogsSenderConfig.Since, "--all", fmt.Sprintf("TAG=%s", tag)).
				Return("Dummy", 0)
		}
		for _, service := range config.LogsSenderConfig.Services {
			outputPath := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", service))
			logsSenderMock.On("ExecuteOutputToFile", outputPath, "journalctl", "-D", "/var/log/journal/",
				"--since", config.LogsSenderConfig.Since, "--all", "-u", service).
				Return("Dummy", 0)
		}
	}

	archiveSuccess := func() {
		logsSenderMock.On("Execute", "tar", "-czvf", archivePath, "-C", filepath.Dir(logsTmpFilesDir),
			filepath.Base(logsTmpFilesDir)).
			Return("Dummy", "", 0)
	}

	AfterEach(func() {
		logsSenderMock.AssertExpectations(GinkgoT())
	})

	It("CreateFolderIfNotExist failed", func() {
		logsSenderMock.On("CreateFolderIfNotExist", logsTmpFilesDir).
			Return(errors.Errorf("Dummy"))
		err := SendLogs(logsSenderMock)
		fmt.Println(err)
		Expect(err).To(HaveOccurred())
	})

	It("ExecuteOutputToFile failed", func() {
		folderSuccess()
		outputPath := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", config.LogsSenderConfig.Tags[0]))
		logsSenderMock.On("ExecuteOutputToFile", outputPath, "journalctl", "-D", "/var/log/journal/",
			"--since", config.LogsSenderConfig.Since, "--all", fmt.Sprintf("TAG=%s", config.LogsSenderConfig.Tags[0])).
			Return("Dummy", -1)
		err := SendLogs(logsSenderMock)
		fmt.Println(err)
		Expect(err).To(HaveOccurred())
	})

	It("Archive failed", func() {
		folderSuccess()
		executeOutputToFileSuccess()
		logsSenderMock.On("Execute", "tar", "-czvf", archivePath, "-C", filepath.Dir(logsTmpFilesDir),
			filepath.Base(logsTmpFilesDir)).
			Return("Dummy", "Dummy", -1)
		err := SendLogs(logsSenderMock)
		fmt.Println(err)
		Expect(err).To(HaveOccurred())
	})

	It("Upload failed", func() {
		folderSuccess()
		executeOutputToFileSuccess()
		archiveSuccess()
		config.LogsSenderConfig.IsBootstrap = true
		config.LogsSenderConfig.InstallerGatherlogging = true
		logsSenderMock.On("ExecutePrivilege", "/usr/local/bin/installer-gather.sh").Return("Dummy", "", 0)
		logsSenderMock.On("ExecutePrivilege", "mv", "/root/log-bundle-.tar.gz", fmt.Sprintf("%s/installer_gather.tar.gz", logsTmpFilesDir)).Return("Dummy", "", 0)
		logsSenderMock.On("FileUploader", archivePath, strfmt.UUID(config.LogsSenderConfig.ClusterID),
			strfmt.UUID(config.LogsSenderConfig.HostID), config.LogsSenderConfig.TargetURL, config.LogsSenderConfig.PullSecretToken, config.GlobalAgentConfig.AgentVersion).
			Return(errors.Errorf("Dummy"))

		err := SendLogs(logsSenderMock)
		fmt.Println(err)
		Expect(err).To(HaveOccurred())
	})

	It("Happy flow", func() {

		folderSuccess()
		executeOutputToFileSuccess()
		archiveSuccess()
		config.LogsSenderConfig.IsBootstrap = true
		config.LogsSenderConfig.InstallerGatherlogging = true
		logsSenderMock.On("ExecutePrivilege", "/usr/local/bin/installer-gather.sh").Return("Dummy", "", 0)
		logsSenderMock.On("ExecutePrivilege", "mv", "/root/log-bundle-.tar.gz", fmt.Sprintf("%s/installer_gather.tar.gz", logsTmpFilesDir)).Return("Dummy", "", 0)
		logsSenderMock.On("FileUploader", archivePath, strfmt.UUID(config.LogsSenderConfig.ClusterID),
			strfmt.UUID(config.LogsSenderConfig.HostID), config.LogsSenderConfig.TargetURL, config.LogsSenderConfig.PullSecretToken, config.GlobalAgentConfig.AgentVersion).
			Return(nil)
		err := SendLogs(logsSenderMock)
		fmt.Println(err)
		Expect(err).NotTo(HaveOccurred())
	})

})
