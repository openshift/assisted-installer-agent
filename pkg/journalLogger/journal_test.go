package journalLogger

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/ssgreg/journald"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Journal Logging test", func() {
	var (
		journalWriter *MockIJournalWriter
		logger        *logrus.Logger
		fields        = map[string]interface{}{
			"TAG": "agent",
		}
	)
	BeforeEach(func() {
		journalWriter = new(MockIJournalWriter)
		logger = logrus.New()
	})

	It("Journal logging", func() {
		logger.AddHook(NewJournalHook(journalWriter, fields))
		journalWriter.On("Send", mock.Anything, journald.PriorityInfo, fields).Return(nil).Times(2)
		journalWriter.On("Send", mock.Anything, journald.PriorityWarning, fields).Return(nil).Times(3)
		journalWriter.On("Send", mock.Anything, journald.PriorityErr, fields).Return(nil).Times(4)

		for i := 0; i != 2; i++ {
			logger.Infof("Info")
		}
		for i := 0; i != 3; i++ {
			logger.Warn("Warning")
		}
		for i := 0; i != 4; i++ {
			logger.Error("Error")
		}
	})
	AfterEach(func() {
		journalWriter.AssertExpectations(GinkgoT())
	})
})

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Logging unit tests")
}
