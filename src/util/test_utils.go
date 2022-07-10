package util

import (
	"net"

	"github.com/stretchr/testify/mock"
)

func FillInterfaceMock(mock *mock.Mock, mtu int, name string, macAddr string, flags net.Flags, addrs []string, speedMbps int64, interfaceType string) {
	mock.On("Name").Return(name)
	mock.On("MTU").Return(mtu)
	hwAddr, _ := net.ParseMAC(macAddr)
	mock.On("HardwareAddr").Return(hwAddr)
	mock.On("Flags").Return(flags)
	mock.On("Addrs").Return(parseAddresses(addrs), nil).Once()
	mock.On("SpeedMbps").Return(speedMbps)
	mock.On("Type").Return(interfaceType, nil).Once()
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
