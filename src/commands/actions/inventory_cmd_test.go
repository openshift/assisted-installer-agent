package actions

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

func verifyPaths(command string, paths []string) {
	for _, path := range paths {
		Expect(command).To(ContainSubstring(fmt.Sprintf("-v %[1]v:%[1]v", path)))
	}
}

var _ = Describe("inventory", func() {
	var hostId strfmt.UUID

	BeforeEach(func() {
		hostId = strfmt.UUID(uuid.New().String())
	})

	It("inventory cmd", func() {

		action, err := New(models.StepTypeInventory, []string{hostId.String()})
		Expect(err).NotTo(HaveOccurred())

		command, args := action.CreateCmd()
		By("running two commands via sh")
		Expect(command).To(Equal("sh"))
		Expect(args[0]).To(Equal("-c"))
		Expect(args[1]).To(ContainSubstring("&&"))

		mtabFile := fmt.Sprintf("/root/mtab-%s", hostId)
		mtabCopy := fmt.Sprintf("cp /etc/mtab %s", mtabFile)
		mtabMount := fmt.Sprintf("%s:/host/etc/mtab:ro", mtabFile)

		Expect(args[1]).To(ContainSubstring(mtabCopy))

		By("verifying mounts to host's filesystem")
		Expect(args[1]).To(ContainSubstring(mtabMount))
		paths := []string{
			"/proc/meminfo",
			"/sys/kernel/mm/hugepages",
			"/proc/cpuinfo",
			"/sys/block",
			"/sys/devices",
			"/sys/bus",
			"/sys/class",
			"/run/udev",
		}
		for _, path := range paths {
			Expect(args[1]).To(ContainSubstring(fmt.Sprintf("-v %[1]v:/host%[1]v:ro", path)))
		}
	})

	It("inventory cmd wrong args number", func() {
		badParamsCommonTests(models.StepTypeDhcpLeaseAllocate, []string{hostId.String()})
	})
})
