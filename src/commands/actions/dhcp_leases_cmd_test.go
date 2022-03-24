package actions

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("dhcp leases", func() {
	var param string

	BeforeEach(func() {
		param = "{\"api_vip_mac\":\"00:1a:4a:5d:6d:90\",\"ingress_vip_mac\":\"00:1a:4a:c9:05:a9\",\"interface\":\"ens3\"}"
	})

	It("dhcp leases", func() {
		action, err := New(models.StepTypeDhcpLeaseAllocate, []string{param})
		Expect(err).NotTo(HaveOccurred())

		command, args := action.CreateCmd()
		Expect(command).To(Equal(podman))
		paths := []string{
			"/var/log",
			"/run/systemd/journal/socket",
		}
		verifyPaths(strings.Join(args, " "), paths)
		Expect(args[len(args)-1]).To(ContainSubstring(param))
	})

	It("dhcp leases wrong args number", func() {
		badParamsCommonTests(models.StepTypeDhcpLeaseAllocate, []string{param})
	})
})