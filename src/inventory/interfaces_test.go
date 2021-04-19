package inventory

import (
	"errors"
	"net"

	"golang.org/x/sys/unix"

	"github.com/stretchr/testify/mock"
	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
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

func newFilledInterfaceMock(mtu int, name string, macAddr string, flags net.Flags, addrs []string, isPhysical bool, isBonding bool, isVlan bool, speedMbps int64) *MockInterface {
	hwAddr, _ := net.ParseMAC(macAddr)
	ret := newInterfaceMock()
	ret.On("IsPhysical").Return(isPhysical)
	if isPhysical || isBonding || isVlan {
		ret.On("Name").Return(name)
		ret.On("MTU").Return(mtu)
		ret.On("HardwareAddr").Return(hwAddr)
		ret.On("Flags").Return(flags)
		ret.On("Addrs").Return(toAddresses(addrs), nil).Once()
		ret.On("SpeedMbps").Return(speedMbps)
	}
	if !isPhysical {
		ret.On("IsBonding").Return(isBonding)
	}
	if !(isPhysical || isBonding) {
		ret.On("IsVlan").Return(isVlan)
	}

	return ret
}

var _ = Describe("Interfaces", func() {
	var dependencies *util.MockIDependencies

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
		interfaceMock := newFilledInterfaceMock(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.18/24", "fe80::d832:8def:dd51:3527/128", "de90::d832:8def:dd51:3527/128"}, true, false, false, 1000)
		dependencies.On("Interfaces").Return([]util.Interface{interfaceMock}, nil).Once()
		dependencies.On("Execute", "biosdevname", "-i", "eth0").Return("em2", "", 0).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth0/carrier").Return([]byte("1\n"), nil).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth0/device/device").Return([]byte("my-device"), nil).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth0/device/vendor").Return([]byte("my-vendor"), nil).Once()
		dependencies.On("LinkByName", "eth0").Return(&netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: "eth0"}}, nil).Once()
		dependencies.On("RouteList", mock.Anything, mock.Anything).Return([]netlink.Route{
			{
				Dst:      &net.IPNet{IP: net.ParseIP("de90::"), Mask: net.CIDRMask(64, 128)},
				Protocol: unix.RTPROT_RA,
			},
		}, nil)
		ret := GetInterfaces(dependencies)
		Expect(len(ret)).To(Equal(1))
		Expect(ret).To(Equal([]*models.Interface{
			{
				Biosdevname:   "em2",
				Flags:         []string{"up", "broadcast"},
				HasCarrier:    true,
				IPV4Addresses: []string{"10.0.0.18/24"},
				IPV6Addresses: []string{"de90::d832:8def:dd51:3527/64"},
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
		rets := []util.Interface{
			newFilledInterfaceMock(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.18/24", "192.168.6.7/20", "fe80::d832:8def:dd51:3527/128", "de90::d832:8def:dd51:3527/128"}, true, false, false, 100),
			newFilledInterfaceMock(1400, "eth1", "f8:75:a4:a4:00:ff", net.FlagBroadcast|net.FlagLoopback, []string{"10.0.0.19/24", "192.168.6.8/20", "fe80::d832:8def:dd51:3528/127", "de90::d832:8def:dd51:3528/127"}, true, false, false, 10),
			newFilledInterfaceMock(1400, "eth2", "f8:75:a4:a4:00:ff", net.FlagBroadcast|net.FlagLoopback, []string{"10.0.0.20/24", "192.168.6.9/20", "fe80::d832:8def:dd51:3529/126", "de90::d832:8def:dd51:3529/126"}, false, false, false, 5),
			newFilledInterfaceMock(1400, "bond0", "f8:75:a4:a4:00:fd", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.21/24", "192.168.6.10/20", "fe80::d832:8def:dd51:3529/125", "de90::d832:8def:dd51:3529/125"}, false, true, false, -1),
			newFilledInterfaceMock(1400, "eth2.10", "f8:75:a4:a4:00:fc", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.25/24", "192.168.6.14/20", "fe80::d832:8def:dd51:3520/125", "de90::d832:8def:dd51:3520/125"}, false, false, true, -1),
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
		dependencies.On("Execute", "biosdevname", "-i", "bond0").Return("bond0", "", 0).Once()
		dependencies.On("ReadFile", "/sys/class/net/bond0/carrier").Return(nil, errors.New("Blah")).Once()
		dependencies.On("ReadFile", "/sys/class/net/bond0/device/device").Return(nil, errors.New("Blah")).Once()
		dependencies.On("ReadFile", "/sys/class/net/bond0/device/vendor").Return([]byte("my-vendor2"), nil).Once()
		dependencies.On("Execute", "biosdevname", "-i", "eth2.10").Return("vlan", "", 0).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth2.10/carrier").Return(nil, errors.New("Blah")).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth2.10/device/device").Return(nil, errors.New("Blah")).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth2.10/device/vendor").Return([]byte("my-vendor2"), nil).Once()
		dependencies.On("LinkByName", mock.Anything).Return(&netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: "eth0"}}, nil)
		dependencies.On("RouteList", mock.Anything, mock.Anything).Return([]netlink.Route{
			{
				Dst:      &net.IPNet{IP: net.ParseIP("de90::"), Mask: net.CIDRMask(62, 128)},
				Protocol: unix.RTPROT_RA,
			},
		}, nil)
		ret := GetInterfaces(dependencies)
		Expect(len(ret)).To(Equal(4))
		Expect(ret).To(Equal([]*models.Interface{
			{
				Biosdevname:   "em2",
				ClientID:      "",
				Flags:         []string{"up", "broadcast"},
				HasCarrier:    false,
				IPV4Addresses: []string{"10.0.0.18/24", "192.168.6.7/20"},
				IPV6Addresses: []string{"de90::d832:8def:dd51:3527/62"},
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
				IPV6Addresses: []string{"de90::d832:8def:dd51:3528/62"},
				MacAddress:    "f8:75:a4:a4:00:ff",
				Mtu:           1400,
				Name:          "eth1",
				Product:       "",
				Vendor:        "my-vendor2",
				SpeedMbps:     10,
			},
			{
				Biosdevname:   "bond0",
				ClientID:      "",
				Flags:         []string{"up", "broadcast"},
				HasCarrier:    false,
				IPV4Addresses: []string{"10.0.0.21/24", "192.168.6.10/20"},
				IPV6Addresses: []string{
					"de90::d832:8def:dd51:3529/62",
				},
				MacAddress: "f8:75:a4:a4:00:fd",
				Mtu:        1400,
				Name:       "bond0",
				Product:    "",
				SpeedMbps:  -1,
				Vendor:     "my-vendor2",
			},
			{
                                Biosdevname:   "vlan",
                                ClientID:      "",
                                Flags:         []string{"up", "broadcast"},
                                HasCarrier:    false,
                                IPV4Addresses: []string{"10.0.0.25/24", "192.168.6.14/20"},
                                IPV6Addresses: []string{
                                        "de90::d832:8def:dd51:3520/62",
                                },
                                MacAddress: "f8:75:a4:a4:00:fc",
                                Mtu:        1400,
                                Name:       "eth2.10",
                                Product:    "",
                                SpeedMbps:  -1,
                                Vendor:     "my-vendor2",
                        },
		}))
		for _, i := range rets {
			i.(*MockInterface).AssertExpectations(GinkgoT())
		}
	})
})
