package inventory

import (
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const MaxIpmiChannel = 12

// bmc handles BMC (Baseboard Management Controller) information retrieval via ipmitool.
//
// BMC commands may fail when running as non-root due to:
//   - ipmitool requiring /dev/ipmi* access (typically root-only)
//   - dhclient requiring raw socket capabilities (CAP_NET_RAW)
//
// These failures are logged at debug level and handled gracefully by returning
// fallback values (0.0.0.0 for IPv4, ::/0 for IPv6). BMC information will be
// unavailable in non-root mode, which is acceptable for most deployment scenarios
// where out-of-band management is not required.
type bmc struct {
	dependicies      util.IDependencies
	subprocessConfig *config.SubprocessConfig
}

func newBMC(subprocessConfig *config.SubprocessConfig, dependencies util.IDependencies) *bmc {
	return &bmc{dependicies: dependencies, subprocessConfig: subprocessConfig}
}

func (b *bmc) getIpForChannnel(ch int) string {
	o, e, exitCode := b.dependicies.Execute("ipmitool", "lan", "print", strconv.FormatInt(int64(ch), 10))
	if exitCode != 0 || strings.HasPrefix(e, "Invalid channel") {
		if exitCode != 0 && !strings.HasPrefix(e, "Invalid channel") {
			logrus.Debugf("ipmitool lan print for channel %d failed (exit code %d): %s", ch, exitCode, e)
		}
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
	if b.subprocessConfig.DryRunEnabled {
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
	// ipmitool is non-critical; return fallback if BMC address unavailable
	logrus.Debug("Could not retrieve BMC IPv4 address via ipmitool, using fallback 0.0.0.0")
	return "0.0.0.0"
}

func GetBmcAddress(subprocessConfig *config.SubprocessConfig, dependencies util.IDependencies) string {
	return newBMC(subprocessConfig, dependencies).getBmcAddress()
}

func (b *bmc) getV6Address(ch int, addressType string) string {
	o, e, exitCode := b.dependicies.Execute("ipmitool", "lan6", "print", strconv.FormatInt(int64(ch), 10), addressType+"_addr")
	if exitCode != 0 {
		logrus.Debugf("ipmitool lan6 print for channel %d (%s_addr) failed (exit code %d): %s", ch, addressType, exitCode, e)
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
	o, e, exitCode := b.dependicies.Execute("ipmitool", "lan6", "print", strconv.FormatInt(int64(ch), 10), "enables")
	if exitCode != 0 {
		logrus.Debugf("ipmitool lan6 print for channel %d (enables) failed (exit code %d): %s", ch, exitCode, e)
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
	if b.subprocessConfig.DryRunEnabled {
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
	// ipmitool is non-critical; return fallback if BMC address unavailable
	logrus.Debug("Could not retrieve BMC IPv6 address via ipmitool, using fallback ::/0")
	return "::/0"
}

func GetBmcV6Address(subprocessConfig *config.SubprocessConfig, dependencies util.IDependencies) string {
	return newBMC(subprocessConfig, dependencies).getBmcV6Address()
}
