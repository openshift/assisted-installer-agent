package inventory

import (
	"errors"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

func newInterfaceMock() *MockInterface {
	return &MockInterface{}
}

func str2Addr(addrStr string) net.Addr {
	ip, ipnet, err := net.ParseCIDR(addrStr)
	if err != nil {
		return &net.IPNet{}
	}
	return &net.IPNet{IP: ip, Mask: ipnet.Mask}
}

func toAddresses(addrs []string) []net.Addr {
	ret := make([]net.Addr, 0)
	for _, a := range addrs {
		ret = append(ret, str2Addr(a))
	}
	return ret
}

func newFilledInterfaceMock(mtu int, name string, macAddr string, flags net.Flags, addrs []string, isPhysical bool, speedMbps int64) *MockInterface {
	hwAddr, _ := net.ParseMAC(macAddr)
	ret := newInterfaceMock()
	ret.On("IsPhysical").Return(isPhysical)
	if isPhysical {
		ret.On("MTU").Return(mtu)
		ret.On("Name").Return(name)
		ret.On("HardwareAddr").Return(hwAddr)
		ret.On("Flags").Return(flags)
		ret.On("Addrs").Return(toAddresses(addrs), nil).Once()
		ret.On("SpeedMbps").Return(speedMbps)
	}
	return ret
}

var _ = Describe("Interfaces", func() {
	var dependencies *MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("Interfaces error", func() {
		dependencies.On("Interfaces").Return(nil, errors.New("Just an error"))
		ret := GetInterfaces(dependencies)
		Expect(len(ret)).To(BeZero())
	})

	It("Empty result", func() {
		dependencies.On("Interfaces").Return(nil, nil)
		ret := GetInterfaces(dependencies)
		Expect(len(ret)).To(BeZero())
	})

	It("Single result", func() {
		interfaceMock := newFilledInterfaceMock(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.18/24", "fe80::d832:8def:dd51:3527/64"}, true, 1000)
		dependencies.On("Interfaces").Return([]Interface{interfaceMock}, nil).Once()
		dependencies.On("Execute", "biosdevname", "-i", "eth0").Return("em2", "", 0).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth0/carrier").Return([]byte("1\n"), nil).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth0/device/device").Return([]byte("my-device"), nil).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth0/device/vendor").Return([]byte("my-vendor"), nil).Once()
		ret := GetInterfaces(dependencies)
		Expect(len(ret)).To(Equal(1))
		Expect(ret).To(Equal([]*models.Interface{
			{
				Biosdevname:   "em2",
				Flags:         []string{"up", "broadcast"},
				HasCarrier:    true,
				IPV4Addresses: []string{"10.0.0.18/24"},
				IPV6Addresses: []string{"fe80::d832:8def:dd51:3527/64"},
				MacAddress:    "f8:75:a4:a4:00:fe",
				Mtu:           1500,
				Name:          "eth0",
				Product:       "my-device",
				Vendor:        "my-vendor",
				SpeedMbps:     1000,
			}}))
		interfaceMock.AssertExpectations(GinkgoT())
	})
	It("Multiple results", func() {
		rets := []Interface{
			newFilledInterfaceMock(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.18/24", "192.168.6.7/20", "fe80::d832:8def:dd51:3527/64"}, true, 100),
			newFilledInterfaceMock(1400, "eth1", "f8:75:a4:a4:00:ff", net.FlagBroadcast|net.FlagLoopback, []string{"10.0.0.19/24", "192.168.6.8/20", "fe80::d832:8def:dd51:3528/64"}, true, 10),
			newFilledInterfaceMock(1400, "eth2", "f8:75:a4:a4:00:ff", net.FlagBroadcast|net.FlagLoopback, []string{"10.0.0.20/24", "192.168.6.9/20", "fe80::d832:8def:dd51:3529/64"}, false, 5),
		}
		dependencies.On("Interfaces").Return(rets, nil).Once()
		dependencies.On("Execute", "biosdevname", "-i", "eth0").Return("em2", "", 0).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth0/carrier").Return([]byte("0\n"), nil).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth0/device/device").Return([]byte("my-device1"), nil).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth0/device/vendor").Return([]byte("my-vendor1"), nil).Once()
		dependencies.On("Execute", "biosdevname", "-i", "eth1").Return("em3", "", 0).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth1/carrier").Return(nil, errors.New("Blah")).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth1/device/device").Return(nil, errors.New("Blah")).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth1/device/vendor").Return([]byte("my-vendor2"), nil).Once()
		ret := GetInterfaces(dependencies)
		Expect(len(ret)).To(Equal(2))
		Expect(ret).To(Equal([]*models.Interface{
			{
				Biosdevname:   "em2",
				ClientID:      "",
				Flags:         []string{"up", "broadcast"},
				HasCarrier:    false,
				IPV4Addresses: []string{"10.0.0.18/24", "192.168.6.7/20"},
				IPV6Addresses: []string{"fe80::d832:8def:dd51:3527/64"},
				MacAddress:    "f8:75:a4:a4:00:fe",
				Mtu:           1500,
				Name:          "eth0",
				Product:       "my-device1",
				Vendor:        "my-vendor1",
				SpeedMbps:     100,
			},
			{
				Biosdevname:   "em3",
				ClientID:      "",
				Flags:         []string{"broadcast", "loopback"},
				HasCarrier:    false,
				IPV4Addresses: []string{"10.0.0.19/24", "192.168.6.8/20"},
				IPV6Addresses: []string{"fe80::d832:8def:dd51:3528/64"},
				MacAddress:    "f8:75:a4:a4:00:ff",
				Mtu:           1400,
				Name:          "eth1",
				Product:       "",
				Vendor:        "my-vendor2",
				SpeedMbps:     10,
			},
		}))
		for _, i := range rets {
			i.(*MockInterface).AssertExpectations(GinkgoT())
		}
	})
})
