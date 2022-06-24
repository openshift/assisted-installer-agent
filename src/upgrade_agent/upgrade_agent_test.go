package upgrade_agent

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	mock "github.com/stretchr/testify/mock"
)

func TestUnitests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Upgrade agent unit tests")
}

var _ = Describe("Upgrade agent command", func() {
	var (
		deps *MockDependencies
		log  *logrus.Logger
	)

	BeforeEach(func() {
		deps = &MockDependencies{}
		log = logrus.New()
		log.SetOutput(GinkgoWriter)
	})

	AfterEach(func() {
		deps.AssertExpectations(GinkgoT())
	})

	It("Succeeds if pull succeeds", func() {
		deps.On(
			"ExecutePrivileged",
			"podman", "pull", "quay.io/my/image:v1.2.3",
		).Return("", "", 0).Once()
		stdout, stderr, code := Run(
			`{ "agent_image": "quay.io/my/image:v1.2.3" }`,
			deps,
			log,
		)
		Expect(code).To(BeZero())
		Expect(stdout).To(MatchJSON(`{
			"agent_image": "quay.io/my/image:v1.2.3",
			"result": "success"
		}`))
		Expect(stderr).To(BeEmpty())
	})

	It("Fails if pull fails", func() {
		deps.On(
			"ExecutePrivileged",
			"podman", "pull", "quay.io/my/image:v1.2.3",
		).Return("", "", 1).Once()
		stdout, stderr, code := Run(
			`{ "agent_image": "quay.io/my/image:v1.2.3" }`,
			deps,
			log,
		)
		Expect(code).To(Equal(1))
		Expect(stdout).To(MatchJSON(`{
			"agent_image": "quay.io/my/image:v1.2.3",
			"result": "failure"
		}`))
		Expect(stderr).To(BeEmpty())
	})

	It("Does nothing if pull is already in progress", func() {
		// This test will run the command twice. The first time inside a separate goroutine
		// that will simulate the image pull taking a long time, and the second time in the
		// current goroutine to verify that it does nothing. We need to make sure that the
		// first goroutine has started waiting before starting the second one, and that it
		// waits till the end of the test. We use these channels for that.
		startPull := make(chan struct{})
		endPull := make(chan struct{})
		defer close(endPull)

		// Prepare the dependecies mock:
		deps.On(
			"ExecutePrivileged",
			"podman", "pull", "quay.io/my/image:v1.2.3",
		).Run(func(args mock.Arguments) {
			close(startPull)
			<-endPull
		}).Return("", "", 0).Once()

		// Execute the action a first time in a separate goroutine, it will wait till the
		// end of the test:
		go Run(
			`{ "agent_image": "quay.io/my/image:v1.2.3" }`,
			deps,
			log,
		)

		// Wait till the first execution of the action has started pulling the image, and
		// then do the second execution, which should do nothing:
		<-startPull
		stdout, stderr, code := Run(
			`{ "agent_image": "quay.io/my/image:v1.2.3" }`,
			deps,
			log,
		)
		Expect(code).To(BeZero())
		Expect(stdout).To(MatchJSON(`{
			"agent_image": "quay.io/my/image:v1.2.3"
		}`))
		Expect(stderr).To(BeEmpty())
	})
})
