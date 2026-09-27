package utils

import (
	"net"
	"sort"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// AddressFilter is a function type to filter addresses
type AddressFilter func(netlink.Addr) bool

// RouteFilter is a function type to filter routes
type RouteFilter func(netlink.Route) bool

type addressMapFunc func(filter AddressFilter) (map[netlink.Link][]netlink.Addr, error)
type routeMapFunc func(filter RouteFilter) (map[int][]netlink.Route, error)

func getAddrs(filter AddressFilter) (addrMap map[netlink.Link][]netlink.Addr, err error) {
	nlHandle, err := netlink.NewHandle(unix.NETLINK_ROUTE)
	if err != nil {
		return nil, err
	}
	defer nlHandle.Delete()

	links, err := nlHandle.LinkList()
	if err != nil {
		return nil, err
	}

	addrMap = make(map[netlink.Link][]netlink.Addr)
	for _, link := range links {
		addresses, err := nlHandle.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			return nil, err
		}
		for _, address := range addresses {
			if filter != nil && !filter(address) {
				log.Debugf("Ignoring filtered address %+v", address)
				continue
			}

			if _, ok := addrMap[link]; ok {
				addrMap[link] = append(addrMap[link], address)
			} else {
				addrMap[link] = []netlink.Addr{address}
			}
		}
	}
	log.Debugf("retrieved Address map %+v", addrMap)
	return addrMap, nil
}

func getRouteMap(filter RouteFilter) (routeMap map[int][]netlink.Route, err error) {
	nlHandle, err := netlink.NewHandle(unix.NETLINK_ROUTE)
	if err != nil {
		return nil, err
	}
	defer nlHandle.Delete()

	routes, err := nlHandle.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return nil, err
	}

	routeMap = make(map[int][]netlink.Route)
	for _, route := range routes {
		if filter != nil && !filter(route) {
			log.Debugf("Ignoring filtered route %+v", route)
			continue
		}
		if _, ok := routeMap[route.LinkIndex]; ok {
			routeMap[route.LinkIndex] = append(routeMap[route.LinkIndex], route)
		} else {
			routeMap[route.LinkIndex] = []netlink.Route{route}
		}
	}

	log.Debugf("Retrieved route map %+v", routeMap)

	return routeMap, nil
}

// ValidNodeAddress returns true if the address is suitable for a node's primary IP
func ValidNodeAddress(address netlink.Addr) bool {
	// Ignore link-local addresses
	if address.IP.IsLinkLocalUnicast() {
		return false
	}

	// Ignore deprecated IPv6 addresses
	if net.IPv6len == len(address.IP) && address.PreferedLft == 0 {
		return false
	}

	return true
}

// usableIPv6Route returns true if the passed route is acceptable for AddressesRouting
func usableIPv6Route(route netlink.Route) bool {
	// Ignore default routes
	if route.Dst == nil {
		return false
	}
	// Ignore non-IPv6 routes
	if net.IPv6len != len(route.Dst.IP) {
		return false
	}
	// Ignore non-advertised routes
	if route.Protocol != unix.RTPROT_RA {
		return false
	}

	return true
}

func isIPv6(ip net.IP) bool {
	return ip.To4() == nil
}

// AddressesRouting takes a slice of Virtual IPs and returns a configured address in the current network namespace that directly routes to at least one of those vips. If the interface containing that address is dual-stack, it will also return a single address of the opposite IP family. You can optionally pass an AddressFilter to further filter down which addresses are considered
func AddressesRouting(vips []net.IP, af AddressFilter) ([]net.IP, error) {
	return addressesRoutingInternal(vips, af, getAddrs, getRouteMap)
}

func addressesRoutingInternal(vips []net.IP, af AddressFilter, getAddrs addressMapFunc, getRouteMap routeMapFunc) ([]net.IP, error) {
	addrMap, err := getAddrs(af)
	if err != nil {
		return nil, err
	}

	var routeMap map[int][]netlink.Route
	matches := make([]net.IP, 0)
	for link, addresses := range addrMap {
	addrLoop:
		for _, address := range addresses {
			isVip := false
			for _, vip := range vips {
				if address.IP.String() == vip.String() {
					log.Debugf("Address %s is VIP %s. Skipping.", address, vip)
					isVip = true
				}
			}
			if isVip {
				continue
			}
			maskPrefix, maskBits := address.Mask.Size()
			if net.IPv6len == len(address.IP) && maskPrefix == maskBits {
				if routeMap == nil {
					routeMap, err = getRouteMap(usableIPv6Route)
					if err != nil {
						return nil, err
					}
				}
				if routes, ok := routeMap[link.Attrs().Index]; ok {
					for _, route := range routes {
						log.Debugf("Checking route %+v (mask %s) for address %+v", route, route.Dst.Mask, address)
						containmentNet := net.IPNet{IP: address.IP, Mask: route.Dst.Mask}
						for _, vip := range vips {
							log.Debugf("Checking whether address %s with route %s contains VIP %s", address, route, vip)
							if containmentNet.Contains(vip) {
								log.Debugf("Address %s with route %s contains VIP %s", address, route, vip)
								matches = append(matches, address.IP)
								break addrLoop
							}
						}
					}
				}
			} else {
				for _, vip := range vips {
					log.Debugf("Checking whether address %s contains VIP %s", address, vip)
					if address.Contains(vip) {
						log.Debugf("Address %s contains VIP %s", address, vip)
						matches = append(matches, address.IP)
						break addrLoop
					}
				}
			}
		}

		if len(matches) > 0 {
			// Find an address of the opposite IP family on the same interface
			for _, address := range addresses {
				if isIPv6(address.IP) != isIPv6(matches[0]) {
					matches = append(matches, address.IP)
					break
				}
			}
			break
		}
	}
	return matches, nil
}

// defaultRoute returns true if the passed route is a default route
func defaultRoute(route netlink.Route) bool {
	return route.Dst == nil
}

// AddressesDefault returns a slice of configured addresses in the current network namespace associated with default routes; IPv4 first (if any), then IPv6 (if any). You can optionally pass an AddressFilter to further filter down which addresses are considered
func AddressesDefault(preferIPv6 bool, af AddressFilter) ([]net.IP, error) {
	return addressesDefaultInternal(preferIPv6, af, getAddrs, getRouteMap)
}

type FoundAddress struct {
	Address   net.IP
	Priority  int
	LinkIndex int
}

func addressesDefaultInternal(preferIPv6 bool, af AddressFilter, getAddrs addressMapFunc, getRouteMap routeMapFunc) ([]net.IP, error) {
	addrMap, err := getAddrs(af)
	if err != nil {
		return nil, err
	}
	routeMap, err := getRouteMap(defaultRoute)
	if err != nil {
		return nil, err
	}

	matches := make([]net.IP, 0)
	addrs := make([]FoundAddress, 0)
	for link, addresses := range addrMap {
		linkIndex := link.Attrs().Index
		if routeMap[linkIndex] == nil {
			continue
		}
		for _, address := range addresses {
			log.Debugf("Address %s is on interface %s with default route", address, link.Attrs().Name)
			// We should only have one default route per interface
			addrs = append(addrs, FoundAddress{
				Address:   address.IP,
				Priority:  routeMap[linkIndex][0].Priority,
				LinkIndex: linkIndex,
			})
		}
	}

	// Sort addresses into a stable order, based on default route priority and link
	// index. Otherwise the order of the addresses we return may change if an address
	// moves to a bridge (for example).
	sort.SliceStable(addrs, func(i, j int) bool {
		if addrs[i].Priority == addrs[j].Priority {
			if addrs[i].LinkIndex == addrs[j].LinkIndex {
				return isIPv6(addrs[i].Address) == preferIPv6 && isIPv6(addrs[j].Address) != preferIPv6
			}
			return addrs[i].LinkIndex < addrs[j].LinkIndex
		}
		return addrs[i].Priority < addrs[j].Priority
	})

	foundv4 := false
	foundv6 := false
	for _, addr := range addrs {
		if (isIPv6(addr.Address) && foundv6) || (!isIPv6(addr.Address) && foundv4) {
			continue
		}
		matches = append(matches, addr.Address)
		if isIPv6(addr.Address) {
			foundv6 = true
		} else {
			foundv4 = true
		}
	}
	return matches, nil
}
