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
	mock "github.com/stretchr/testify/mock"
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
		config.LogsSenderConfig.IsBootstrap = true
		config.LogsSenderConfig.InstallerGatherlogging = true
	})

	folderSuccess := func() {
		logsSenderMock.On("CreateFolderIfNotExist", logsTmpFilesDir).
			Return(nil)
	}

	executeOutputToFileSuccess := func(retVal int) {
		for _, tag := range config.LogsSenderConfig.Tags {
			outputPath := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", tag))
			logsSenderMock.On("ExecuteOutputToFile", outputPath, "journalctl", "-D", "/var/log/journal/",
				"--since", config.LogsSenderConfig.Since, "--all", fmt.Sprintf("TAG=%s", tag)).
				Return("Dummy", retVal)
		}
		for _, service := range config.LogsSenderConfig.Services {
			outputPath := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", service))
			logsSenderMock.On("ExecuteOutputToFile", outputPath, "journalctl", "-D", "/var/log/journal/",
				"--since", config.LogsSenderConfig.Since, "--all", "-u", service).
				Return("Dummy", retVal)
		}
	}

	archiveSuccess := func() {
		logsSenderMock.On("Execute", "tar", "-czvf", archivePath, "-C", filepath.Dir(logsTmpFilesDir),
			filepath.Base(logsTmpFilesDir)).
			Return("Dummy", "", 0)
	}

	fileUploaderSuccess := func() {
		logsSenderMock.On("FileUploader", archivePath, strfmt.UUID(config.LogsSenderConfig.ClusterID),
			strfmt.UUID(config.LogsSenderConfig.HostID), config.LogsSenderConfig.TargetURL, config.LogsSenderConfig.PullSecretToken, config.GlobalAgentConfig.AgentVersion).
			Return(nil)
	}

	gatherInstallerLogsSuccess := func() {
		logsSenderMock.On("GatherInstallerLogs", mock.Anything).Return(nil)
	}

	gatherErrorLogsSuccess := func() {
		logsSenderMock.On("GatherErrorLogs", mock.Anything).Return(nil)
	}

	AfterEach(func() {
		logsSenderMock.AssertExpectations(GinkgoT())
	})

	It("CreateFolderIfNotExist failed", func() {
		logsSenderMock.On("CreateFolderIfNotExist", logsTmpFilesDir).
			Return(errors.Errorf("Dummy"))
		err, report := SendLogs(logsSenderMock)
		fmt.Println(err)
		Expect(err).To(HaveOccurred())
		Expect(report).To(Equal(""))
	})

	It("GatherInstallerLogs failed", func() {
		folderSuccess()
		gatherErrorLogsSuccess()
		executeOutputToFileSuccess(0)
		archiveSuccess()
		fileUploaderSuccess()
		logsSenderMock.On("GatherInstallerLogs", logsTmpFilesDir).Return(errors.New("Dummy"))
		err, report := SendLogs(logsSenderMock)
		Expect(err).To(Not(HaveOccurred()))
		Expect(report).To(ContainSubstring("Dummy"))
	})

	It("GatherErrorLogs failed", func() {
		folderSuccess()
		gatherInstallerLogsSuccess()
		executeOutputToFileSuccess(0)
		archiveSuccess()
		fileUploaderSuccess()
		logsSenderMock.On("GatherErrorLogs", logsTmpFilesDir).Return(errors.New("Dummy"))
		err, report := SendLogs(logsSenderMock)
		Expect(err).NotTo(HaveOccurred())
		Expect(report).To(ContainSubstring("Dummy"))
	})

	It("ExecuteOutputToFile failed", func() {
		config.LogsSenderConfig.InstallerGatherlogging = false
		folderSuccess()
		archiveSuccess()
		fileUploaderSuccess()
		executeOutputToFileSuccess(-1)
		err, report := SendLogs(logsSenderMock)
		Expect(err).To(Not(HaveOccurred()))
		Expect(report).To(ContainSubstring("Dummy"))
	})

	It("Archive failed", func() {
		folderSuccess()
		gatherInstallerLogsSuccess()
		gatherErrorLogsSuccess()
		executeOutputToFileSuccess(0)
		logsSenderMock.On("Execute", "tar", "-czvf", archivePath, "-C", filepath.Dir(logsTmpFilesDir),
			filepath.Base(logsTmpFilesDir)).
			Return("Dummy", "Dummy", -1)
		err, _ := SendLogs(logsSenderMock)
		Expect(err).To(HaveOccurred())
	})

	It("Upload failed", func() {
		folderSuccess()
		gatherInstallerLogsSuccess()
		gatherErrorLogsSuccess()
		executeOutputToFileSuccess(0)
		archiveSuccess()
		logsSenderMock.On("FileUploader", archivePath, strfmt.UUID(config.LogsSenderConfig.ClusterID),
			strfmt.UUID(config.LogsSenderConfig.HostID), config.LogsSenderConfig.TargetURL, config.LogsSenderConfig.PullSecretToken, config.GlobalAgentConfig.AgentVersion).
			Return(errors.Errorf("Dummy"))
		err, _ := SendLogs(logsSenderMock)
		fmt.Println(err)
		Expect(err).To(HaveOccurred())
	})

	It("dmseg logs", func() {
		outputPath := path.Join(logsTmpFilesDir, "dmesg.logs")
		logsSenderMock.On("ExecuteOutputToFile", outputPath, "dmesg").Return("Dummy", 0)
		err := getDmesgLogs(logsSenderMock, outputPath)
		Expect(err).NotTo(HaveOccurred())
	})

	It("core dump logs", func() {
		outputPath := path.Join(logsTmpFilesDir, "coredump_exe_bc_pid_5459")
		logsSenderMock.On("ExecutePrivileged","coredumpctl", "list", "--no-legend").
		Return("Thu 2019-11-07 15:14:46 CET    5459  1000  1000   6 present   /usr/bin/bc\n", "", 0)
		logsSenderMock.On("ExecutePrivileged", "coredumpctl", "dump", "5459", "--output", outputPath).Return("","",0)
		err := getCoreDumps(logsSenderMock, logsTmpFilesDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("full journal logs", func() {
		outputPath := path.Join(logsTmpFilesDir, "journal.logs")
		logsSenderMock.On("ExecuteOutputToFile", outputPath, "journalctl", "-D", "/var/log/journal/",
		"--since", config.LogsSenderConfig.Since, "--all").Return("Dummy", 0)
		err := getJournalLogs(logsSenderMock, config.LogsSenderConfig.Since, outputPath, []string{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("Happy flow", func() {
		folderSuccess()
		gatherInstallerLogsSuccess()
		gatherErrorLogsSuccess()
		executeOutputToFileSuccess(0)
		archiveSuccess()
		fileUploaderSuccess()
		err, _ := SendLogs(logsSenderMock)
		fmt.Println(err)
		Expect(err).NotTo(HaveOccurred())
	})
})
