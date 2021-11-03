package inventory

import (
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"gopkg.in/yaml.v2"
)

const MaxIpmiChannel = 12

type bmc struct {
	dependicies util.IDependencies
}

func newBMC(dependencies util.IDependencies) *bmc {
	return &bmc{dependicies: dependencies}
}

func (b *bmc) getIpForChannnel(ch int) string {
	o, e, exitCode := b.dependicies.Execute("ipmitool", "lan", "print", strconv.FormatInt(int64(ch), 10))
	if exitCode != 0 || strings.HasPrefix(e, "Invalid channel") {
		return ""
	}
	r := regexp.MustCompile("^IP Address[ \t]*:[ \t]*([^ \t]*)[ \t]*$")
	for _, line := range strings.Split(o, "\n") {
		matches := r.FindStringSubmatch(line)
		if len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

func (b *bmc) getIsEnabled(value interface{}) bool {
	return value != false && value != ""
}

func (b *bmc) getBmcAddress() string {
	if config.GlobalDryRunConfig.DryRunEnabled {
		// This action is too slow and unnecessary, so skip it in dry run
		return "0.0.0.0"
	}

	for ch := 1; ch <= MaxIpmiChannel; ch++ {
		ret := b.getIpForChannnel(ch)
		if ret == "" {
			continue
		}
		ip := net.ParseIP(ret)
		if ip == nil {
			continue
		}
		if ret != "0.0.0.0" {
			return ret
		}
	}
	return "0.0.0.0"
}

func GetBmcAddress(dependencies util.IDependencies) string {
	return newBMC(dependencies).getBmcAddress()
}

func (b *bmc) getV6Address(ch int, addressType string) string {
	o, _, exitCode := b.dependicies.Execute("ipmitool", "lan6", "print", strconv.FormatInt(int64(ch), 10), addressType+"_addr")
	if exitCode != 0 {
		return ""
	}
	m := make(map[interface{}]interface{})
	if err := yaml.Unmarshal([]byte(o), &m); err != nil {
		return ""
	}
	nullAddressRE := regexp.MustCompile(`^::(/\d{1,3})*$`)
	for _, v := range m {
		addressMap, ok := v.(map[interface{}]interface{})
		if !ok {
			continue
		}
		addressValue, ok := addressMap["Address"]
		if !ok {
			continue
		}
		address := addressValue.(string)
		var enabled bool
		if addressType == "dynamic" {
			st, ok := addressMap["Source/Type"]
			if !ok {
				continue
			}
			switch st {
			case "DHCPv6", "SLAAC":
				enabled = true
			}
		} else {
			value, ok := addressMap["Enabled"]
			if ok {
				enabled = b.getIsEnabled(value)
			}
		}
		status, ok := addressMap["Status"]
		if ok && status == "active" && enabled && !nullAddressRE.MatchString(address) {
			return address
		}
	}
	return ""
}

func (b *bmc) getAddrMode(ch int) string {
	o, _, exitCode := b.dependicies.Execute("ipmitool", "lan6", "print", strconv.FormatInt(int64(ch), 10), "enables")
	if exitCode != 0 {
		return ""
	}
	r := regexp.MustCompile("^IPv6/IPv4 Addressing Enables: (both|ipv6)[ \t]*$")
	for _, line := range strings.Split(o, "\n") {
		matches := r.FindStringSubmatch(line)
		if len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

func (b *bmc) getBmcV6Address() string {
	if config.GlobalDryRunConfig.DryRunEnabled {
		// This action is too slow and unnecessary, so skip it in dry run
		return "::/0"
	}

	for ch := 1; ch <= MaxIpmiChannel; ch++ {
		addrMode := b.getAddrMode(ch)
		if addrMode == "" {
			continue
		}
		address := b.getV6Address(ch, "dynamic")
		if address == "" {
			address = b.getV6Address(ch, "static")
		}
		if address == "" {
			continue
		}
		ip, _, err := net.ParseCIDR(address)
		if err != nil {
			continue
		}
		return ip.String()
	}
	return "::/0"
}

func GetBmcV6Address(dependencies util.IDependencies) string {
	return newBMC(dependencies).getBmcV6Address()
}
