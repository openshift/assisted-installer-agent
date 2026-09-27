package util

import (
	"net"

	"github.com/stretchr/testify/mock"
	"github.com/vishvananda/netlink"
)

func NewFilledMockInterface(mtu int, name string, macAddr string, flags net.Flags, addrs []string, speedMbps int64, interfaceType string) *MockInterface {
	interfaceMock := MockInterface{}
	FillInterfaceMock(&interfaceMock.Mock, mtu, name, macAddr, flags, addrs, speedMbps, interfaceType)
	return &interfaceMock
}

func FillInterfaceMock(mock *mock.Mock, mtu int, name string, macAddr string, flags net.Flags, addrs []string, speedMbps int64, interfaceType string) {
	mock.On("Name").Return(name)
	mock.On("MTU").Return(mtu)
	hwAddr, _ := net.ParseMAC(macAddr)
	mock.On("HardwareAddr").Return(hwAddr).Maybe()
	mock.On("Flags").Return(flags)
	mock.On("Addrs").Return(parseAddresses(addrs), nil).Once()
	mock.On("SpeedMbps").Return(speedMbps)
	mock.On("Type").Return(interfaceType, nil).Once()
	mock.On("IsPhysical").Return(interfaceType == "physical").Maybe()
	mock.On("IsBonding").Return(interfaceType == "bond").Maybe()
	mock.On("IsVlan").Return(interfaceType == "vlan").Maybe()

}

func parseAddresses(addrs []string) []net.Addr {
	ret := make([]net.Addr, 0)
	for _, a := range addrs {
		ret = append(ret, parseAddress(a))
	}
	return ret
}

func parseAddress(addrStr string) net.Addr {
	ip, ipnet, err := net.ParseCIDR(addrStr)
	if err != nil {
		return &net.IPNet{}
	}
	return &net.IPNet{IP: ip, Mask: ipnet.Mask}
}

func SetupNetlinkMocks(dependencies *MockIDependencies, interfaces []Interface) {
	for _, iface := range interfaces {
		name := iface.Name()
		hwAddr := iface.HardwareAddr()

		link := &netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{
				Name:         name,
				HardwareAddr: hwAddr,
			},
		}
		dependencies.On("LinkByName", name).Return(link, nil).Maybe()
	}
}
