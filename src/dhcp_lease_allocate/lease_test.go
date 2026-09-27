package dhcp_lease_allocate

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	singleLease = `lease {
		single Lease;
}`
	firstLease = `lease {
		first Lease;
}`
	secondLease = `lease {
		second Lease;
}`
	twoLeases = firstLease + "\n" + secondLease
)

var _ = Describe("Extract lease", func() {
	var dependencies *MockDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	Context("extractLastLease", func() {
		It("Single lease", func() {
			dependencies.On("ReadFile", "Blah").Return([]byte(singleLease), nil).Once()
			lastLease, err := extractLastLease(dependencies, "Blah")
			Expect(err).ToNot(HaveOccurred())
			Expect(lastLease).To(Equal(singleLease))
		})

		It("Untrimmed single lease", func() {
			dependencies.On("ReadFile", "Blah").Return([]byte("\n\t \n"+singleLease+" \t\n\n "), nil).Once()
			lastLease, err := extractLastLease(dependencies, "Blah")
			Expect(err).ToNot(HaveOccurred())
			Expect(lastLease).To(Equal(singleLease))
		})

		It("Invalid single lease", func() {
			dependencies.On("ReadFile", "Blah").Return([]byte("\n\t \n"+singleLease+" \t\n\n l"), nil).Once()
			_, err := extractLastLease(dependencies, "Blah")
			Expect(err).To(HaveOccurred())
		})
		It("Two leases", func() {
			dependencies.On("ReadFile", "Blah").Return([]byte(twoLeases), nil).Once()
			lastLease, err := extractLastLease(dependencies, "Blah")
			Expect(err).ToNot(HaveOccurred())
			Expect(lastLease).To(Equal(secondLease))
		})
	})
})
