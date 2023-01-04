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
		interfaceMock := util.NewMockInterface(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.18/24", "fe80::d832:8def:dd51:3527/128", "de90::d832:8def:dd51:3527/128"}, 1000, "physical")
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
				Type:          "physical",
			}}))
		interfaceMock.AssertExpectations(GinkgoT())
	})
	It("Multiple results", func() {
		rets := []util.Interface{
			util.NewMockInterface(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.18/24", "192.168.6.7/20", "fe80::d832:8def:dd51:3527/128", "de90::d832:8def:dd51:3527/128"}, 100, "physical"),
			util.NewMockInterface(1400, "eth1", "f8:75:a4:a4:00:ff", net.FlagBroadcast|net.FlagLoopback, []string{"10.0.0.19/24", "192.168.6.8/20", "fe80::d832:8def:dd51:3528/127", "de90::d832:8def:dd51:3528/127"}, 10, "physical"),
			util.NewMockInterface(1400, "eth2", "f8:75:a4:a4:00:ff", net.FlagBroadcast|net.FlagLoopback, []string{"10.0.0.20/24", "192.168.6.9/20", "fe80::d832:8def:dd51:3529/126", "de90::d832:8def:dd51:3529/126"}, 5, ""),
			util.NewMockInterface(1400, "bond0", "f8:75:a4:a4:00:fd", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.21/24", "192.168.6.10/20", "fe80::d832:8def:dd51:3529/125", "de90::d832:8def:dd51:3529/125"}, -1, "bond"),
			util.NewMockInterface(1400, "eth2.10", "f8:75:a4:a4:00:fc", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.25/24", "192.168.6.14/20", "fe80::d832:8def:dd51:3520/125", "de90::d832:8def:dd51:3520/125"}, -1, "vlan"),
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
		dependencies.On("Execute", "biosdevname", "-i", "eth2").Return("em4", "", 0).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth2/carrier").Return(nil, errors.New("Blah")).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth2/device/device").Return(nil, errors.New("Blah")).Once()
		dependencies.On("ReadFile", "/sys/class/net/eth2/device/vendor").Return([]byte("my-vendor2"), nil).Once()
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
		Expect(len(ret)).To(Equal(5))
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
				Type:          "physical",
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
				Type:          "physical",
			},
			{
				Biosdevname:   "em4",
				ClientID:      "",
				Flags:         []string{"broadcast", "loopback"},
				HasCarrier:    false,
				IPV4Addresses: []string{"10.0.0.20/24", "192.168.6.9/20"},
				IPV6Addresses: []string{"de90::d832:8def:dd51:3529/62"},
				MacAddress:    "f8:75:a4:a4:00:ff",
				Mtu:           1400,
				Name:          "eth2",
				Product:       "",
				SpeedMbps:     5,
				Type:          "",
				Vendor:        "my-vendor2",
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
				Type:       "bond",
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
				Type:       "vlan",
			},
		}))
		for _, i := range rets {
			i.(*util.MockInterface).AssertExpectations(GinkgoT())
		}
	})
})
