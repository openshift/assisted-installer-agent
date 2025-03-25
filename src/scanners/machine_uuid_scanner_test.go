package scanners

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	agentutils "github.com/openshift/assisted-installer-agent/src/util"

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
	var dependencies *agentutils.MockIDependencies

	BeforeEach(func() {
		dependencies = &agentutils.MockIDependencies{}
		serialDiscovery = &MockSerialDiscovery{}
	})

	AfterEach(func() {
		serialDiscovery.AssertExpectations(GinkgoT())
	})

	It("Empty serial", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{}, nil).Once()
		serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: TestUuid}, nil)
		id := ReadId(serialDiscovery, dependencies)
		Expect(id).To(Equal(toUUID(TestUuid)))
	})
	It("Unknown serial", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: util.UNKNOWN}, nil).Once()
		serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: TestUuid}, nil)
		id := ReadId(serialDiscovery, dependencies)
		Expect(id).To(Equal(toUUID(TestUuid)))
	})
	It("Vmware None serial", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: "None"}, nil).Once()
		serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: TestUuid}, nil)
		id := ReadId(serialDiscovery, dependencies)
		Expect(id).To(Equal(toUUID(TestUuid)))
	})
	It("unspecified serial", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: SerialUnspecifiedBaseBoardString}, nil).Once()
		serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: TestUuid}, nil)
		id := ReadId(serialDiscovery, dependencies)
		Expect(id).To(Equal(toUUID(TestUuid)))
	})
	It("System x3850 X5 (none) serial", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: "(none)"}, nil).Once()
		serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: TestUuid}, nil)
		id := ReadId(serialDiscovery, dependencies)
		Expect(id).To(Equal(toUUID(TestUuid)))
	})
	It("dash serial", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: "-"}, nil).Once()
		serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: TestUuid}, nil)
		id := ReadId(serialDiscovery, dependencies)
		Expect(id).To(Equal(toUUID(TestUuid)))
	})
	It("dash serial embedded", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: "123-456-789"}, nil).Once()
		id := ReadId(serialDiscovery, dependencies)
		Expect(id).To(Equal(toUUID("856f4f9c-3c08-4d97-8ec7-ea0ad7d4cadf")))
	})

	tests := []struct {
		useCase  string
		mbSerial string
		uuid     string
	}{
		{useCase: "kaloom", mbSerial: SerialDefaultString, uuid: KaloomUUID},
		{useCase: "proliantgen11", mbSerial: SerialProliantGen11, uuid: ZeroesUUID},
		{useCase: "zeroes", mbSerial: SerialDefaultString, uuid: ZeroesUUID},
		{useCase: "linode", mbSerial: SerialNotSpecified, uuid: "Not Settable"},
	}

	for i := range tests {
		test := tests[i]

		It(fmt.Sprintf("mac address fallback %s", test.useCase), func() {
			rets := []agentutils.Interface{
				newMockInterface(65536, "lo", "", net.FlagBroadcast|net.FlagLoopback, []string{"127.0.0.1/8"}, 100, "physical"),
				newMockInterface(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.18/24", "192.168.6.7/20", "fe80::d832:8def:dd51:3527/128", "de90::d832:8def:dd51:3527/128"}, 100, "physical"),
				newMockInterface(1400, "eth1", "f8:75:a4:a4:00:ff", net.FlagBroadcast|net.FlagLoopback, []string{"10.0.0.19/24", "192.168.6.8/20", "fe80::d832:8def:dd51:3528/127", "de90::d832:8def:dd51:3528/127"}, 10, "physical"),
			}
			dependencies.On("Interfaces").Return(rets, nil).Once()
			dependencies.On("Execute", "biosdevname", "-i", "lo").Return("em2", "", 0).Once()
			dependencies.On("ReadFile", "/sys/class/net/lo/carrier").Return([]byte("0\n"), nil).Once()
			dependencies.On("ReadFile", "/sys/class/net/lo/device/device").Return([]byte("my-device1"), nil).Once()
			dependencies.On("ReadFile", "/sys/class/net/lo/device/vendor").Return([]byte("my-vendor1"), nil).Once()
			dependencies.On("Execute", "biosdevname", "-i", "eth0").Return("em2", "", 0).Once()
			dependencies.On("ReadFile", "/sys/class/net/eth0/carrier").Return([]byte("0\n"), nil).Once()
			dependencies.On("ReadFile", "/sys/class/net/eth0/device/device").Return([]byte("my-device1"), nil).Once()
			dependencies.On("ReadFile", "/sys/class/net/eth0/device/vendor").Return([]byte("my-vendor1"), nil).Once()
			dependencies.On("Execute", "biosdevname", "-i", "eth1").Return("em3", "", 0).Once()
			dependencies.On("ReadFile", "/sys/class/net/eth1/carrier").Return(nil, errors.New("Blah")).Once()
			dependencies.On("ReadFile", "/sys/class/net/eth1/device/device").Return(nil, errors.New("Blah")).Once()
			dependencies.On("ReadFile", "/sys/class/net/eth1/device/vendor").Return([]byte("my-vendor2"), nil).Once()
			dependencies.On("LinkByName", mock.Anything).Return(&netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: "eth0"}}, nil)
			dependencies.On("RouteList", mock.Anything, mock.Anything).Return([]netlink.Route{
				{
					Dst:      &net.IPNet{IP: net.ParseIP("de90::"), Mask: net.CIDRMask(64, 128)},
					Protocol: unix.RTPROT_RA,
				},
			}, nil)

			serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: test.mbSerial}, nil).Once()
			serialDiscovery.On("Product").Return(&ghw.ProductInfo{UUID: test.uuid}, nil)
			id := ReadId(serialDiscovery, dependencies)
			Expect(id).To(Equal(md5GenerateUUID("f8:75:a4:a4:00:fe")))
		})
	}
	It("Other", func() {
		serialDiscovery.On("Baseboard").Return(&ghw.BaseboardInfo{SerialNumber: "Other"}, nil).Once()
		id := ReadId(serialDiscovery, dependencies)
		Expect(id).To(Equal(toUUID("6311ae17-c1ee-52b3-6e68-aaf4ad066387")))
	})
})

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scanner unit tests")
}

func newMockInterface(mtu int, name string, macAddr string, flags net.Flags, addrs []string, speedMbps int64, interfaceType string) *agentutils.MockInterface {
	interfaceMock := agentutils.MockInterface{}
	agentutils.FillInterfaceMock(&interfaceMock.Mock, mtu, name, macAddr, flags, addrs, speedMbps, interfaceType)
	return &interfaceMock
}
