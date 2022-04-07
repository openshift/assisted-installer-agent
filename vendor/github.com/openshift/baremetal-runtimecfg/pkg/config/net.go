package config

import (
	"fmt"
	"net"
	"strings"

	"github.com/openshift/baremetal-runtimecfg/pkg/utils"
)

func getInterfaceAndNonVIPAddr(vips []net.IP) (vipIface net.Interface, nonVipAddr *net.IPNet, err error) {
	if len(vips) < 1 {
		return vipIface, nonVipAddr, fmt.Errorf("At least one VIP needs to be fed to this function")
	}
	vipMap := make(map[string]net.IP)
	for _, vip := range vips {
		vipMap[vip.String()] = vip
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return vipIface, nonVipAddr, err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return vipIface, nonVipAddr, err
		}
		for _, addr := range addrs {
			switch n := addr.(type) {
			case *net.IPNet:
				if _, ok := vipMap[n.IP.String()]; ok {
					continue // This is a VIP, let's skip
				}
				_, nn, _ := net.ParseCIDR(strings.Replace(addr.String(), "/128", "/64", 1))

				if nn.Contains(vips[0]) {
					// Since IPV6 subnet is set to /64 we should also verify that
					// the candidate address and VIP address are L2 connected.
					// To make sure that the correct interface being chosen for cases like:
					// 2 interfaces , subnetA: 1001:db8::/120 , subnetB: 1001:db8::f00/120 and VIP address  1001:db8::64
					nodeAddrs, err := utils.AddressesRouting(vips, utils.ValidNodeAddress)
					if err == nil && len(nodeAddrs) > 0 && n.IP.Equal(nodeAddrs[0]) {
						return iface, n, nil
					}
				}
			default:
				fmt.Println("not supported addr")
			}
		}
	}

	nodeAddrs, err := utils.AddressesDefault(false, utils.ValidNodeAddress)
	if err != nil {
		return vipIface, nonVipAddr, err
	}
	if len(nodeAddrs) == 0 {
		return vipIface, nonVipAddr, fmt.Errorf("No interface nor address found")
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return vipIface, nonVipAddr, err
		}
		for _, addr := range addrs {
			switch n := addr.(type) {
			case *net.IPNet:
				if n.IP.String() == nodeAddrs[0].String() {
					return iface, n, nil
				}
			default:
				fmt.Println("not supported addr")
			}
		}
	}
	return vipIface, nonVipAddr, fmt.Errorf("No interface nor address found")
}
