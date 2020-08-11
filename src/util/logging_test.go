package util

import (
	"io"
	"io/ioutil"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/pkg/journalLogger"
	"github.com/sirupsen/logrus"
	"github.com/ssgreg/journald"
	"github.com/stretchr/testify/mock"
)

type WriterMock struct {
	mock.Mock
}

func (w *WriterMock) Write(p []byte) (n int, err error) {
	ret := w.Called(p)
	return ret.Int(0), ret.Error(1)
}

var _ = Describe("Logging test", func() {
	var (
		writer        *WriterMock
		journalWriter *journalLogger.MockIJournalWriter
		discard       *WriterMock
		logger        *logrus.Logger
		fields        = map[string]interface{}{
			"TAG": "agent",
		}
	)
	BeforeEach(func() {
		writer = new(WriterMock)
		journalWriter = new(journalLogger.MockIJournalWriter)
		discard = new(WriterMock)
		ioutil.Discard = discard
		getLogFileWriter = func(name string) (io.Writer, error) {
			return writer, nil
		}
		logger = logrus.New()
	})

	It("Text logging", func() {
		writer.On("Write", mock.Anything).Return(5, nil)
		setLogging(logger, journalWriter, "agent", true, false)
		logger.Infof("Hello")
	})
	It("Both", func() {
		writer.On("Write", mock.Anything).Return(5, nil)
		journalWriter.On("Send", mock.Anything, journald.PriorityInfo, fields).Return(nil).Once()
		setLogging(logger, journalWriter, "agent", true, true)
		logger.Infof("Hello1")
	})
	It("None", func() {
		discard.On("Write", mock.Anything).Return(5, nil).Once()
		setLogging(logger, journalWriter, "agent", false, false)
		logger.Infof("Hello2")
	})
	AfterEach(func() {
		writer.AssertExpectations(GinkgoT())
		journalWriter.AssertExpectations(GinkgoT())
		discard.AssertExpectations(GinkgoT())
	})
})

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Logging unit tests")
}
