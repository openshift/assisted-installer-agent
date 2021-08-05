package inventory

import (
	"net"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

const (
	familyIPv4 int = 4
	familyIPv6 int = 6
)

var (
	familyDelimiter map[int]string = map[int]string{familyIPv4: ".", familyIPv6: ":"}
)

type handler interface {
	getRouteList() ([]netlink.Route, error)
	getLinkName(netlink.Route) (string, error)
	getFamily() int
}

type routeHandler struct {
	family int
}

func (rh routeHandler) getRouteList() ([]netlink.Route, error) {
	return netlink.RouteList(nil, rh.family)
}

func (rh routeHandler) getLinkName(route netlink.Route) (string, error) {
	linkIndex := route.LinkIndex
	if len(route.MultiPath) > 0 {
		linkIndex = route.MultiPath[0].LinkIndex // get the index of the first element.
	}
	link, err := netlink.LinkByIndex(linkIndex)
	if err != nil {
		return "", err
	}
	return link.Attrs().Name, nil
}

func (rh routeHandler) getFamily() int {
	return rh.family
}

func sameFamily(route netlink.Route, family int) bool {
	if route.Dst == nil && route.Gw == nil && len(route.MultiPath) == 0 {
		return true
	}
	if route.Dst != nil && strings.Contains(route.Dst.IP.String(), familyDelimiter[family]) {
		return true
	}

	if route.Gw != nil && strings.Contains(route.Gw.String(), familyDelimiter[family]) {
		return true
	}

	if len(route.MultiPath) > 0 {
		if route.MultiPath[0].NewDst != nil && family == route.MultiPath[0].NewDst.Family() {
			return true
		}
		if route.MultiPath[0].Gw != nil && strings.Contains(route.MultiPath[0].Gw.String(), familyDelimiter[family]) {
			return true
		}
	}
	return false
}

func getIPZero(family int) *net.IP {
	if family == familyIPv4 {
		return &net.IPv4zero
	}
	return &net.IPv6zero
}
func GetRoutes(dependencies util.IDependencies) []*models.Route {

	rh4 := routeHandler{family: familyIPv4}
	rh6 := routeHandler{family: familyIPv6}
	routes, err := getIPRoutes(rh4)
	if err != nil {
		logrus.Warnf("Unable to determine the IPv4 routes: %s", err)
	}
	ipv6Routes, err := getIPRoutes(rh6)
	if err != nil {
		logrus.Warnf("Unable to determine the IPv6 routes: %s", err)
		return routes
	}
	routes = append(routes, ipv6Routes...)
	return routes
}

func getIPRoutes(h handler) ([]*models.Route, error) {
	rList, err := h.getRouteList()
	if err != nil {
		logrus.Errorf("Unable to retrieve the IPv%d routes: %s", h.getFamily(), err)
		return []*models.Route{}, err
	}
	routes := []*models.Route{}
	for _, r := range rList {
		if !sameFamily(r, h.getFamily()) {
			continue
		}
		linkName, err := h.getLinkName(r)
		if err != nil {
			logrus.Errorf("Unable to retrieve the link name for index %d: %s", r.LinkIndex, err)
			return nil, err
		}
		var dst, gw string
		if r.Dst == nil {
			dst = getIPZero(h.getFamily()).String()
		} else {
			dst = r.Dst.IP.String()
		}
		if len(r.MultiPath) > 0 && r.MultiPath[0].Gw != nil {
			gw = r.MultiPath[0].Gw.String()
		} else if r.Gw != nil {
			gw = r.Gw.String()
		}
		routes = append(routes, &models.Route{
			Interface:   linkName,
			Destination: dst,
			Gateway:     gw,
			Family:      int32(h.getFamily()),
		})
	}
	return routes, nil
}
