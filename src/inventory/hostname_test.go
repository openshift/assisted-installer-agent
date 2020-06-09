package inventory

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Hostname test", func() {
	var dependencies *MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("Execute error", func() {
		dependencies.On("Hostname").Return("Blah", fmt.Errorf("Just an error")).Once()
		ret := GetHostname(dependencies)
		Expect(ret).To(Equal(""))
	})

	It("Untrimmed hostname", func() {
		dependencies.On("Hostname").Return("\t myhostname.com \n\t", nil).Once()
		ret := GetHostname(dependencies)
		Expect(ret).To(Equal("myhostname.com"))
	})
	It("Happy flow", func() {
		dependencies.On("Hostname").Return("myhostname.com", nil).Once()
		ret := GetHostname(dependencies)
		Expect(ret).To(Equal("myhostname.com"))
	})
})
