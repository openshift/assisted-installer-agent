package util

import (
	"net"
	"strings"
)

// IsIPv4Addr returns true if the input is a valid IPv4 address
func IsIPv4Addr(ip string) bool {
	return strings.Contains(ip, ".") && net.ParseIP(ip) != nil
}
