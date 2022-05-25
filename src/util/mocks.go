package util

import (
	"net"

	netlink "github.com/vishvananda/netlink"
)

//go:generate mockery -name Link -inpkg
type Link interface {
	netlink.Link
}

func NewFilledInterfaceMock(mtu int, name string, macAddr string, flags net.Flags, addrs []string, isPhysical bool, isBonding bool, isVlan bool, speedMbps int64, interfaceType string) *MockInterface {
	hwAddr, _ := net.ParseMAC(macAddr)
	ret := MockInterface{}
	ret.On("Name").Return(name)
	ret.On("MTU").Return(mtu)
	ret.On("HardwareAddr").Return(hwAddr)
	ret.On("Flags").Return(flags)
	ret.On("Addrs").Return(toAddresses(addrs), nil).Once()
	ret.On("SpeedMbps").Return(speedMbps)
	ret.On("Type").Return(interfaceType, nil).Once()

	return &ret
}

func toAddresses(addrs []string) []net.Addr {
	ret := make([]net.Addr, 0)
	for _, a := range addrs {
		ret = append(ret, str2Addr(a))
	}
	return ret
}

func str2Addr(addrStr string) net.Addr {
	ip, ipnet, err := net.ParseCIDR(addrStr)
	if err != nil {
		return &net.IPNet{}
	}
	return &net.IPNet{IP: ip, Mask: ipnet.Mask}
}
