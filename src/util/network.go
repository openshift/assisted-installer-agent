package util

import (
	"net"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// IsIPv4Addr returns true if the input is a valid IPv4 address
func IsIPv4Addr(ip string) bool {
	return strings.Contains(ip, ".") && net.ParseIP(ip) != nil
}

//RouteFinder defines functions needed to find routes by link
//go:generate mockery -name RouteFinder -inpkg
type RouteFinder interface {
	LinkByName(name string) (netlink.Link, error)
	RouteList(link netlink.Link, family int) ([]netlink.Route, error)
}

//NetlinkRouteFinder implements RouteFinder using netling library
type NetlinkRouteFinder struct{}

//LinkByName returns a link by a network interface name, it such interface exists
func (f *NetlinkRouteFinder) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}

//RouteList returns a list of routes filtered by a link and IP family (IPv4/IPv6)
func (f *NetlinkRouteFinder) RouteList(link netlink.Link, family int) ([]netlink.Route, error) {
	return netlink.RouteList(link, family)
}

// SetV6PrefixesForAddress updates a list of IPv6 addresses with values that have a correct CIDR.
// This is needed because otherwise all IPv6 addresses will appear with /128 (single host).
func SetV6PrefixesForAddress(link string, finder RouteFinder, log logrus.FieldLogger, addresses []string) error {

	if len(addresses) == 0 {
		return nil
	}

	l, err := finder.LinkByName(link)
	if err != nil {
		return err
	}

	routes, err := getUsableIPv6Routes(l, finder, log)
	if err != nil {
		return err
	}

	for i, addr := range addresses {

		if addr == "" || !strings.Contains(addr, ":") {
			continue
		}

		ip, _, err := net.ParseCIDR(addr)
		if err != nil {
			log.WithError(err).Warnf("Error parsing CIDR %s", addr)
			continue
		}

		for _, route := range routes {
			containmentNet := net.IPNet{IP: route.Dst.IP, Mask: route.Dst.Mask}
			if containmentNet.Contains(ip) {
				addresses[i] = (&net.IPNet{IP: ip, Mask: route.Dst.Mask}).String()
				break
			}
		}
	}

	return nil
}

func getUsableIPv6Routes(link netlink.Link, finder RouteFinder, log logrus.FieldLogger) ([]netlink.Route, error) {

	routes, err := finder.RouteList(link, netlink.FAMILY_V6)
	if err != nil {
		return nil, err
	}

	usableRoutes := make([]netlink.Route, 0)
	for _, route := range routes {
		if !isUsableIPv6Route(route) {
			log.Debugf("Ignoring filtered route %+v", route)
			continue
		}
		usableRoutes = append(usableRoutes, route)
	}

	return usableRoutes, nil
}

// isUsableIPv6Route returns true if the passed route is acceptable for AddressesRouting
func isUsableIPv6Route(route netlink.Route) bool {
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
