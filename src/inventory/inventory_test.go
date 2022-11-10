package inventory

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("Hostname processing", func() {
	DescribeTable(
		"Rename behaviour",
		func(original string, interfaces []*models.Interface, expected string) {
			// Create the inventory:
			inventory := &models.Inventory{
				Hostname: original,
			}
			inventory.Interfaces = []*models.Interface{{
				Name:          "lo",
				MacAddress:    "",
				IPV4Addresses: []string{"127.0.0.1/32"},
				IPV6Addresses: []string{"::1/128"},
			}}
			inventory.Interfaces = append(inventory.Interfaces, interfaces...)

			// Process the inventory and check the results:
			processInventory(inventory)
			Expect(inventory.Hostname).To(Equal(expected))
		},
		Entry(
			"Doesn't rename if already has acceptable name",
			"myhost",
			[]*models.Interface{
				{
					Name:          "eth0",
					Type:          "physical",
					MacAddress:    "71:A0:A4:6F:BE:C8",
					IPV4Addresses: []string{"192.168.0.1/24"},
				},
			},
			"myhost",
		),
		Entry(
			"Replaces IPv4 localhost with MAC of first NIC",
			"localhost",
			[]*models.Interface{
				{
					Name:          "eth0",
					Type:          "physical",
					MacAddress:    "71:A0:A4:6F:BE:C8",
					IPV4Addresses: []string{"192.168.0.1/24"},
				},
			},
			"71-a0-a4-6f-be-c8",
		),
		Entry(
			"Replace IPv6 localhost with MAC of first NIC",
			"localhost6",
			[]*models.Interface{
				{
					Name:          "eth0",
					Type:          "physical",
					MacAddress:    "71:A0:A4:6F:BE:C8",
					IPV4Addresses: []string{"192.168.0.1/24"},
				},
			},
			"71-a0-a4-6f-be-c8",
		),
		Entry(
			"Ignores NIC with MAC but no IP address",
			"localhost",
			[]*models.Interface{
				{
					Name:          "eth0",
					Type:          "physical",
					MacAddress:    "71:A0:A4:6F:BE:C8",
					IPV4Addresses: []string{},
				},
				{
					Name:          "eth1",
					Type:          "physical",
					MacAddress:    "42:5a:90:c8:24:dc",
					IPV4Addresses: []string{"192.168.0.1/24"},
				},
			},
			"42-5a-90-c8-24-dc",
		),
		Entry(
			"Ignores NIC with loopback IPv4 address",
			"localhost",
			[]*models.Interface{
				{
					Name:          "eth0",
					Type:          "physical",
					MacAddress:    "71:A0:A4:6F:BE:C8",
					IPV4Addresses: []string{"127.0.0.1/32"},
				},
				{
					Name:          "eth1",
					Type:          "physical",
					MacAddress:    "42:5a:90:c8:24:dc",
					IPV4Addresses: []string{"192.168.0.1/24"},
				},
			},
			"42-5a-90-c8-24-dc",
		),
		Entry(
			"Ignores NIC with loopback IPv6 address",
			"localhost",
			[]*models.Interface{
				{
					Name:          "eth0",
					Type:          "physical",
					MacAddress:    "71:A0:A4:6F:BE:C8",
					IPV6Addresses: []string{"::1/128"},
				},
				{
					Name:          "eth1",
					Type:          "physical",
					MacAddress:    "42:5a:90:c8:24:dc",
					IPV4Addresses: []string{"192.168.0.1/24"},
				},
			},
			"42-5a-90-c8-24-dc",
		),
		Entry(
			"Uses NIC with IPv6 address",
			"localhost",
			[]*models.Interface{
				{
					Name:          "eth0",
					Type:          "physical",
					MacAddress:    "71:A0:A4:6F:BE:C8",
					IPV6Addresses: []string{"5dc8:725d:26ae:1192:d336:54a3:d7c7:23a7/64"},
				},
				{
					Name:          "eth1",
					Type:          "physical",
					MacAddress:    "42:5a:90:c8:24:dc",
					IPV4Addresses: []string{"192.168.0.1/24"},
				},
			},
			"71-a0-a4-6f-be-c8",
		),
		Entry(
			"Orders NICs",
			"localhost",
			[]*models.Interface{
				{
					Name:          "b",
					Type:          "physical",
					MacAddress:    "42:5a:90:c8:24:dc",
					IPV4Addresses: []string{"192.168.0.2/24"},
				},
				{
					Name:          "a",
					Type:          "physical",
					MacAddress:    "71:A0:A4:6F:BE:C8",
					IPV4Addresses: []string{"192.168.0.1/24"},
				},
			},
			"71-a0-a4-6f-be-c8",
		),
		Entry(
			"Ignores non physical NIC",
			"localhost",
			[]*models.Interface{
				{
					Name:          "eth0",
					Type:          "virtual",
					MacAddress:    "42:5a:90:c8:24:dc",
					IPV4Addresses: []string{"192.168.0.2/24"},
				},
				{
					Name:          "eth1",
					Type:          "physical",
					MacAddress:    "71:A0:A4:6F:BE:C8",
					IPV4Addresses: []string{"192.168.0.1/24"},
				},
			},
			"71-a0-a4-6f-be-c8",
		),
	)
})
