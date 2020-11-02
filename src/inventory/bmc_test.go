package inventory

import (
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	bmcV4OkAnswer = `Set in Progress         : Set Complete
Auth Type Support       : MD5 
Auth Type Enable        : Callback : MD5 
                        : User     : MD5 
                        : Operator : MD5 
                        : Admin    : MD5 
                        : OEM      : 
IP Address Source       : DHCP Address
IP Address              : 10.16.218.144
Subnet Mask             : 255.255.254.0
MAC Address             : 4c:d9:8f:03:e8:74
SNMP Community String   : public
IP Header               : TTL=0x40 Flags=0x40 Precedence=0x00 TOS=0x10
BMC ARP Control         : ARP Responses Enabled, Gratuitous ARP Disabled
Gratituous ARP Intrvl   : 2.0 seconds
Default Gateway IP      : 10.16.219.254
Default Gateway MAC     : 00:00:00:00:00:00
Backup Gateway IP       : 0.0.0.0
Backup Gateway MAC      : 00:00:00:00:00:00
802.1q VLAN ID          : Disabled
802.1q VLAN Priority    : 0
RMCP+ Cipher Suites     : 0,1,2,3,4,5,6,7,8,9,10,11,12,13,14
Cipher Suite Priv Max   : Xaaaaaaaaaaaaaa
                        :     X=Cipher Suite Unused
                        :     c=CALLBACK
                        :     u=USER
                        :     o=OPERATOR
                        :     a=ADMIN
                        :     O=OEM
Bad Password Threshold  : Not Available
`
	bmcV6NoAddress = `
IPv6/IPv4 Support:
    IPv6 only: yes
    IPv4 and IPv6: yes
    IPv6 Destination Addresses for LAN alerting: yes
IPv6/IPv4 Addressing Enables: ipv4
IPv6 Header Traffic Class: 0
IPv6 Header Static Hop Limit: 0
IPv6 Status:
    Static address max:  1
    Dynamic address max: 16
    DHCPv6 support:      yes
    SLAAC support:       yes
IPv6 Static Address 0:
    Enabled:        no
    Address:        ::/64
    Status:         disabled
IPv6 DHCPv6 Static DUID Storage Length: 3
IPv6 DHCPv6 Static DUID 0:
    Length:   0
    Type:     unknown
IPv6 Dynamic Address 0:
    Source/Type:    static
    Address:        ::/0
    Status:         active
IPv6 Dynamic Address 1:
    Source/Type:    static
    Address:        ::/0
    Status:         active
IPv6 Dynamic Address 2:
    Source/Type:    static
    Address:        ::/0
    Status:         active
IPv6 Dynamic Address 3:
    Source/Type:    static
    Address:        ::/0
    Status:         active
IPv6 DHCPv6 Dynamic DUID Storage Length: 3
IPv6 DHCPv6 Dynamic DUID 0:
    Length:   0
    Type:     unknown
IPv6 DHCPv6 Timing Configuration Support: not supported
IPv6 Router Address Configuration Control:
    Enable static router address:  no
    Enable dynamic router address: no
IPv6 Static Router 1:
    Address: ::
    MAC:     00:00:00:00:00:00
    Prefix:  ::/255
IPv6 Static Router 2:
    Address: ::
    MAC:     00:00:00:00:00:00
    Prefix:  ::/255
IPv6 Number of Dynamic Router Info Sets: 16
IPv6 Dynamic Router 0:
    Address: ::
    MAC:     00:00:00:00:00:00
    Prefix:  ::/0
IPv6 Dynamic Router 1:
    Address: ::
    MAC:     00:00:00:00:00:00
    Prefix:  ::/0
IPv6 Dynamic Router 2:
    Address: ::
    MAC:     00:00:00:00:00:00
    Prefix:  ::/0
IPv6 Dynamic Router 3:
    Address: ::
    MAC:     00:00:00:00:00:00
    Prefix:  ::/0
IPv6 ND/SLAAC Timing Configuration Support: not supported
`

	bmcV6Dynamic = `
IPv6/IPv4 Support:
    IPv6 only: yes
    IPv4 and IPv6: yes
    IPv6 Destination Addresses for LAN alerting: yes
IPv6/IPv4 Addressing Enables: ipv4
IPv6 Header Traffic Class: 0
IPv6 Header Static Hop Limit: 0
IPv6 Status:
    Static address max:  1
    Dynamic address max: 16
    DHCPv6 support:      yes
    SLAAC support:       yes
IPv6 Static Address 0:
    Enabled:        no
    Address:        ::/64
    Status:         disabled
IPv6 DHCPv6 Static DUID Storage Length: 3
IPv6 DHCPv6 Static DUID 0:
    Length:   0
    Type:     unknown
IPv6 Dynamic Address 0:
    Source/Type:    DHCPv6
    Address:        fe80::779e:a22f:dc5e:ca41/64
    Status:         active
IPv6 Dynamic Address 1:
    Source/Type:    static
    Address:        ::/0
    Status:         active
IPv6 Dynamic Address 2:
    Source/Type:    static
    Address:        ::/0
    Status:         active
IPv6 Dynamic Address 3:
    Source/Type:    static
    Address:        ::/0
    Status:         active
IPv6 DHCPv6 Dynamic DUID Storage Length: 3
IPv6 DHCPv6 Dynamic DUID 0:
    Length:   0
    Type:     unknown
IPv6 DHCPv6 Timing Configuration Support: not supported
IPv6 Router Address Configuration Control:
    Enable static router address:  no
    Enable dynamic router address: no
IPv6 Static Router 1:
    MAC:     00:00:00:00:00:00
    Prefix:  ::/255
IPv6 Static Router 2:
    MAC:     00:00:00:00:00:00
    Prefix:  ::/255
IPv6 Number of Dynamic Router Info Sets: 16
IPv6 Dynamic Router 0:
    MAC:     00:00:00:00:00:00
    Prefix:  ::/0
IPv6 Dynamic Router 1:
    MAC:     00:00:00:00:00:00
    Prefix:  ::/0
IPv6 Dynamic Router 2:
    MAC:     00:00:00:00:00:00
    Prefix:  ::/0
IPv6 Dynamic Router 3:
    MAC:     00:00:00:00:00:00
    Prefix:  ::/0
IPv6 ND/SLAAC Timing Configuration Support: not supported
`
	bmcV6Static = `IPv6/IPv4 Support:
    IPv6 only: yes
    IPv4 and IPv6: yes
    IPv6 Destination Addresses for LAN alerting: yes
IPv6/IPv4 Addressing Enables: ipv4
IPv6 Header Traffic Class: 0
IPv6 Header Static Hop Limit: 0
IPv6 Status:
    Static address max:  1
    Dynamic address max: 16
    DHCPv6 support:      yes
    SLAAC support:       yes
IPv6 Static Address 0:
    Enabled:        no
    Address:        ::/64
    Status:         disabled
IPv6 DHCPv6 Static DUID Storage Length: 3
IPv6 DHCPv6 Static DUID 0:
    Length:   0
    Type:     unknown
IPv6 Dynamic Address 0:
    Enabled:        true
    Address:        fe80::779e:a22f:dc5e:ca42/64
    Status:         active
IPv6 Dynamic Address 1:
    Source/Type:    static
    Address:        ::/0
    Status:         active
IPv6 Dynamic Address 2:
    Source/Type:    static
    Address:        ::/0
    Status:         active
IPv6 Dynamic Address 3:
    Source/Type:    static
    Address:        ::/0
    Status:         active
IPv6 DHCPv6 Dynamic DUID Storage Length: 3
IPv6 DHCPv6 Dynamic DUID 0:
    Length:   0
    Type:     unknown
IPv6 DHCPv6 Timing Configuration Support: not supported
IPv6 Router Address Configuration Control:
    Enable static router address:  no
    Enable dynamic router address: no
IPv6 Static Router 1:
    MAC:     00:00:00:00:00:00
    Prefix:  ::/255
IPv6 Static Router 2:
    MAC:     00:00:00:00:00:00
    Prefix:  ::/255
IPv6 Number of Dynamic Router Info Sets: 16
IPv6 Dynamic Router 0:
    MAC:     00:00:00:00:00:00
    Prefix:  ::/0
IPv6 Dynamic Router 1:
    MAC:     00:00:00:00:00:00
    Prefix:  ::/0
IPv6 Dynamic Router 2:
    MAC:     00:00:00:00:00:00
    Prefix:  ::/0
IPv6 Dynamic Router 3:
    MAC:     00:00:00:00:00:00
    Prefix:  ::/0
IPv6 ND/SLAAC Timing Configuration Support: not supported
`
)

var _ = Describe("bmc", func() {
	var dependencies *MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("ipv4 happy flow", func() {
		for i := 1; i != 5; i++ {
			ch := strconv.FormatInt(int64(i), 10)
			dependencies.On("Execute", "ipmitool", "lan", "print", ch).Return("", "Invalid channel 10", 0).Once()
		}
		dependencies.On("Execute", "ipmitool", "lan", "print", "5").Return(bmcV4OkAnswer, "", 0).Once()
		addr := GetBmcAddress(dependencies)
		Expect(addr).To(Equal("10.16.218.144"))
	})
	It("ipv4 not found", func() {
		for i := 1; i != 13; i++ {
			ch := strconv.FormatInt(int64(i), 10)
			dependencies.On("Execute", "ipmitool", "lan", "print", ch).Return("", "Invalid channel 10", 0).Once()
		}
		addr := GetBmcAddress(dependencies)
		Expect(addr).To(Equal("0.0.0.0"))
	})

	It("ipv6 not enabled", func() {
		for i := 1; i != 4; i++ {
			ch := strconv.FormatInt(int64(i), 10)
			dependencies.On("Execute", "ipmitool", "lan6", "print", ch, "enables").Return("", "Failed to get IPv6/IPv4 Addressing Enables: Invalid data field in request", -1).Once()
		}
		dependencies.On("Execute", "ipmitool", "lan6", "print", "4", "enables").Return("IPv6/IPv4 Addressing Enables: ipv4", "", 0).Once()
		for i := 5; i != 13; i++ {
			ch := strconv.FormatInt(int64(i), 10)
			dependencies.On("Execute", "ipmitool", "lan6", "print", ch, "enables").Return("", "Failed to get IPv6/IPv4 Addressing Enables: Invalid data field in request", -1).Once()
		}
		addr := GetBmcV6Address(dependencies)
		Expect(addr).To(Equal("::/0"))
	})

	It("ipv6 enabled not found", func() {
		for i := 1; i != 5; i++ {
			dependencies.On("Execute", "ipmitool", "lan6", "print", strconv.FormatInt(int64(i), 10), "enables").Return("", "Failed to get IPv6/IPv4 Addressing Enables: Invalid data field in request", -1).Once()
		}
		dependencies.On("Execute", "ipmitool", "lan6", "print", "5", "dynamic_addr").Return(bmcV6NoAddress, "", 0).Once()
		dependencies.On("Execute", "ipmitool", "lan6", "print", "5", "static_addr").Return(bmcV6NoAddress, "", 0).Once()
		dependencies.On("Execute", "ipmitool", "lan6", "print", "5", "enables").Return("IPv6/IPv4 Addressing Enables: ipv6", "", 0).Once()
		for i := 6; i != 13; i++ {
			dependencies.On("Execute", "ipmitool", "lan6", "print", strconv.FormatInt(int64(i), 10), "enables").Return("", "Failed to get IPv6/IPv4 Addressing Enables: Invalid data field in request", -1).Once()
		}
		addr := GetBmcV6Address(dependencies)
		Expect(addr).To(Equal("::/0"))
	})

	It("ipv6 dynamic found", func() {
		for i := 1; i != 5; i++ {
			ch := strconv.FormatInt(int64(i), 10)
			dependencies.On("Execute", "ipmitool", "lan6", "print", ch, "enables").Return("", "Failed to get IPv6/IPv4 Addressing Enables: Invalid data field in request", -1).Once()
		}
		dependencies.On("Execute", "ipmitool", "lan6", "print", "5", "enables").Return("IPv6/IPv4 Addressing Enables: both", "", 0).Once()
		dependencies.On("Execute", "ipmitool", "lan6", "print", "5", "dynamic_addr").Return(bmcV6Dynamic, "", 0).Once()
		addr := GetBmcV6Address(dependencies)
		Expect(addr).To(Equal("fe80::779e:a22f:dc5e:ca41"))
	})
	It("ipv6 static found", func() {
		for i := 1; i != 5; i++ {
			ch := strconv.FormatInt(int64(i), 10)
			dependencies.On("Execute", "ipmitool", "lan6", "print", ch, "enables").Return("", "Failed to get IPv6/IPv4 Addressing Enables: Invalid data field in request", -1).Once()
		}
		dependencies.On("Execute", "ipmitool", "lan6", "print", "5", "enables").Return("IPv6/IPv4 Addressing Enables: both", "", 0).Once()
		dependencies.On("Execute", "ipmitool", "lan6", "print", "5", "dynamic_addr").Return(bmcV6NoAddress, "", 0).Once()
		dependencies.On("Execute", "ipmitool", "lan6", "print", "5", "static_addr").Return(bmcV6Static, "", 0).Once()
		addr := GetBmcV6Address(dependencies)
		Expect(addr).To(Equal("fe80::779e:a22f:dc5e:ca42"))
	})
})
