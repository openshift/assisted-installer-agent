package scanners

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	TestUuid = "8a8f14ba-81b0-4a5f-a01c-e1e28c1470ba"
)

func toUUID(s string) *strfmt.UUID {
	ret := strfmt.UUID(s)
	return &ret
}

var _ = Describe("Machine uuid test", func() {
	var serialDiscovery *MockSerialDiscovery

	BeforeEach(func() {
		serialDiscovery = &MockSerialDiscovery{}
	})

	AfterEach(func() {
		serialDiscovery.AssertExpectations(GinkgoT())
	})

	It("Empty serial", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{}, nil).Once()
		serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: TestUuid}, nil)
		id := ReadId(serialDiscovery)
		Expect(id).To(Equal(toUUID(TestUuid)))
	})
	It("Unknown serial", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: util.UNKNOWN}, nil).Once()
		serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: TestUuid}, nil)
		id := ReadId(serialDiscovery)
		Expect(id).To(Equal(toUUID(TestUuid)))
	})
	It("Vmware None", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: VmwareDefaultSerial}, nil).Once()
		serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: TestUuid}, nil)
		id := ReadId(serialDiscovery)
		Expect(id).To(Equal(toUUID(TestUuid)))
	})
	It("Other", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: "Other"}, nil).Once()
		id := ReadId(serialDiscovery)
		Expect(id).To(Equal(toUUID("6311ae17-c1ee-52b3-6e68-aaf4ad066387")))
	})
})

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scanner unit tests")
}
