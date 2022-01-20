package scanners

import (
	"github.com/openshift/assisted-installer-agent/src/inventory"
	agent_utils "github.com/openshift/assisted-installer-agent/src/util"
	"sort"
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
	It("Vmware None serial", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: "None"}, nil).Once()
		serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: TestUuid}, nil)
		id := ReadId(serialDiscovery)
		Expect(id).To(Equal(toUUID(TestUuid)))
	})
	It("unspecified serial", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: "Unspecified Base Board Serial Number"}, nil).Once()
		serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: TestUuid}, nil)
		id := ReadId(serialDiscovery)
		Expect(id).To(Equal(toUUID(TestUuid)))
	})
	It("default string serial and system uuid", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: "Default string"}, nil).Once()
		serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: "Default string"}, nil)
		id := ReadId(serialDiscovery)
		interfaces := inventory.GetInterfaces(agent_utils.NewDependencies(""))
		sort.Slice(interfaces, func(i, j int) bool {
			return interfaces[i].MacAddress < interfaces[j].MacAddress
		})
		Expect(id).To(Equal(md5GenerateUUID(interfaces[0].MacAddress)))
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
