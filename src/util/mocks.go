package util

import (
	netlink "github.com/vishvananda/netlink"
)

//go:generate mockery --name Link --inpackage
type Link interface {
	netlink.Link
}
