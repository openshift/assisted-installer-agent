package logs_sender

import (
	"fmt"
	"path"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	models "github.com/openshift/assisted-service/models"
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
	var loggingConfig *config.LogsSenderConfig

	BeforeEach(func() {
		logsSenderMock = &MockLogsSender{}
		archivePath = fmt.Sprintf("%s/logs.tar.gz", logsDir)
		loggingConfig = &config.LogsSenderConfig{}
		loggingConfig.Tags = []string{"agent", "installer"}
		loggingConfig.Services = []string{"test", "test1"}
		loggingConfig.Since = "5 seconds ago"
		loggingConfig.TargetURL = "http://test.com"
		loggingConfig.PullSecretToken = "test"
		loggingConfig.ClusterID = uuid.New().String()
		loggingConfig.HostID = uuid.New().String()
		loggingConfig.InfraEnvID = uuid.New().String()
		logsTmpFilesDir = path.Join(logsDir, fmt.Sprintf("logs_host_%s", loggingConfig.HostID))
		loggingConfig.IsBootstrap = true
		loggingConfig.InstallerGatherlogging = true
	})

	folderSuccess := func() {
		logsSenderMock.On("CreateFolderIfNotExist", logsTmpFilesDir).
			Return(nil)
	}

	executeOutputToFileSuccess := func(retVal int) {
		for _, tag := range loggingConfig.Tags {
			outputPath := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", tag))
			logsSenderMock.On("ExecuteOutputToFile", outputPath, "journalctl", "-D", "/var/log/journal/", "--all",
				"--since", loggingConfig.Since, fmt.Sprintf("TAG=%s", tag)).
				Return("Dummy", retVal)
		}
		for _, service := range loggingConfig.Services {
			outputPath := path.Join(logsTmpFilesDir, fmt.Sprintf("%s.logs", service))
			logsSenderMock.On("ExecuteOutputToFile", outputPath, "journalctl", "-D", "/var/log/journal/", "--all",
				"--since", loggingConfig.Since, "-u", service).
				Return("Dummy", retVal)
		}
	}

	archiveSuccess := func() {
		logsSenderMock.On("Execute", "tar", "-czvf", archivePath, "-C", filepath.Dir(logsTmpFilesDir),
			filepath.Base(logsTmpFilesDir)).
			Return("Dummy", "", 0)
	}

	fileUploaderSuccess := func() {
		logsSenderMock.On("FileUploader", archivePath).
			Return(nil)
	}

	reportLogProgressSuccess := func(completed, collected bool) {
		logsSenderMock.On("LogProgressReport", models.LogsStateRequested).Return(nil)
		if collected {
			logsSenderMock.On("LogProgressReport", models.LogsStateCollecting).Return(nil)
		}
		if completed {
			logsSenderMock.On("LogProgressReport", models.LogsStateCompleted).Return(nil)
		}
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
		reportLogProgressSuccess(false, false)
		err, report := SendLogs(loggingConfig, logsSenderMock)
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
		reportLogProgressSuccess(true, true)
		logsSenderMock.On("GatherInstallerLogs", logsTmpFilesDir).Return(errors.New("Dummy"))
		err, report := SendLogs(loggingConfig, logsSenderMock)
		Expect(err).To(Not(HaveOccurred()))
		Expect(report).To(ContainSubstring("Dummy"))
	})

	It("GatherErrorLogs failed", func() {
		folderSuccess()
		gatherInstallerLogsSuccess()
		executeOutputToFileSuccess(0)
		archiveSuccess()
		fileUploaderSuccess()
		reportLogProgressSuccess(true, true)
		logsSenderMock.On("GatherErrorLogs", logsTmpFilesDir).Return(errors.New("Dummy"))
		err, report := SendLogs(loggingConfig, logsSenderMock)
		Expect(err).NotTo(HaveOccurred())
		Expect(report).To(ContainSubstring("Dummy"))
	})

	It("ExecuteOutputToFile failed", func() {
		loggingConfig.InstallerGatherlogging = false
		folderSuccess()
		archiveSuccess()
		fileUploaderSuccess()
		reportLogProgressSuccess(true, true)
		executeOutputToFileSuccess(-1)
		err, report := SendLogs(loggingConfig, logsSenderMock)
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
		reportLogProgressSuccess(false, true)
		err, _ := SendLogs(loggingConfig, logsSenderMock)
		Expect(err).To(HaveOccurred())
	})

	It("Upload failed", func() {
		folderSuccess()
		gatherInstallerLogsSuccess()
		gatherErrorLogsSuccess()
		executeOutputToFileSuccess(0)
		archiveSuccess()
		logsSenderMock.On("FileUploader", archivePath).
			Return(errors.Errorf("Dummy"))
		reportLogProgressSuccess(false, true)
		err, _ := SendLogs(loggingConfig, logsSenderMock)
		fmt.Println(err)
		Expect(err).To(HaveOccurred())
	})

	It("dmseg logs", func() {
		outputPath := path.Join(logsTmpFilesDir, "dmesg.logs")
		logsSenderMock.On("ExecuteOutputToFile", outputPath, "dmesg", "-T").Return("Dummy", 0)
		err := getDmesgLogs(logsSenderMock, outputPath)
		Expect(err).NotTo(HaveOccurred())
	})

	It("core dump logs", func() {
		outputPath := path.Join(logsTmpFilesDir, "coredump_exe_bc_pid_5459")
		logsSenderMock.On("ExecutePrivileged", "coredumpctl", "list", "--no-legend").
			Return("Thu 2019-11-07 15:14:46 CET    5459  1000  1000   6 present   /usr/bin/bc\n", "", 0)
		logsSenderMock.On("ExecutePrivileged", "coredumpctl", "dump", "5459", "--output", outputPath).Return("", "", 0)
		err := getCoreDumps(logsSenderMock, logsTmpFilesDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("journal logs with since", func() {
		outputPath := path.Join(logsTmpFilesDir, "journal.logs")
		logsSenderMock.On("ExecuteOutputToFile", outputPath, "journalctl", "-D", "/var/log/journal/", "--all",
			"--since", loggingConfig.Since).Return("Dummy", 0)
		err := getJournalLogs(logsSenderMock, loggingConfig.Since, outputPath, []string{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("full journal logs", func() {
		outputPath := path.Join(logsTmpFilesDir, "journal.logs")
		loggingConfig.Since = ""
		logsSenderMock.On("ExecuteOutputToFile", outputPath, "journalctl", "-D", "/var/log/journal/",
			"--all").Return("Dummy", 0)
		err := getJournalLogs(logsSenderMock, loggingConfig.Since, outputPath, []string{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("Happy flow", func() {
		folderSuccess()
		gatherInstallerLogsSuccess()
		gatherErrorLogsSuccess()
		executeOutputToFileSuccess(0)
		archiveSuccess()
		fileUploaderSuccess()
		reportLogProgressSuccess(true, true)
		err, _ := SendLogs(loggingConfig, logsSenderMock)
		fmt.Println(err)
		Expect(err).NotTo(HaveOccurred())
	})
})
