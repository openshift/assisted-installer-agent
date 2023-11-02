package inventory

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/assisted-installer-agent/src/util"
)

const (
	cmdlineNoPxe   = `BOOT_IMAGE=(hd0,gpt2)/vmlinuz-5.5.17-200.fc31.x86_64 root=/dev/mapper/fedora_dhcp--0--223-root ro resume=/dev/mapper/fedora_dhcp--0--223-swap rd.lvm.lv=fedora_dhcp-0-223/root rd.luks.uuid=luks-47bb99f4-7573-42cf-bfcf-92aaa826fb9b rd.lvm.lv=fedora_dhcp-0-223/swap rhgb quiet psmouse.elantech_smbus=0 systemd.unified_cgroup_hierarchy=0`
	cmdlineWithPxe = `BOOT_IMAGE=(hd0,gpt2)/vmlinuz-5.5.17-200.fc31.x86_64 root=/dev/mapper/fedora_dhcp--0--223-root ro BOOTIF=80:32:53:4f:cf:d6 resume=/dev/mapper/fedora_dhcp--0--223-swap rd.lvm.lv=fedora_dhcp-0-223/root rd.luks.uuid=luks-47bb99f4-7573-42cf-bfcf-92aaa826fb9b rd.lvm.lv=fedora_dhcp-0-223/swap rhgb quiet psmouse.elantech_smbus=0 systemd.unified_cgroup_hierarchy=0`
	cmdlines390x   = `rd.neednet=1 console=ttysclp0 coreos.live.rootfs_url=http://172.3.136.6:8080/assisted-installer/rootfs.img ip=10.14.6.3::10.14.6.1:255.255.255.0:master-0.boea3e06.lnxero1.boe:encbdd0:none nameserver=10.14.6.1 ip=[fd00::3]::[fd00::1]:64::encbdd0:none nameserver=[fd00::1] zfcp.allow_lun_scan=0 rd.znet=qeth,0.0.bdd0,0.0.bdd1,0.0.bdd2,layer2=1 rd.dasd=0.0.5235 rd.dasd=0.0.5236 random.trust_cpu=on rd.luks.options=discard ignition.firstboot ignition.platform.id=metal console=tty1 console=ttyS1,115200n8`
)

var _ = Describe("boot", func() {

	var dependencies *util.MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("pxe+mode fail", func() {
		fileInfoMock := MockFileInfo{}
		fileInfoMock.On("IsDir").Return(true)
		dependencies.On("Stat", "/sys/firmware/efi").Return(&fileInfoMock, fmt.Errorf("Just error")).Once()
		dependencies.On("ReadFile", "/proc/cmdline").Return(nil, fmt.Errorf("Just another error"))
		bootRecord := GetBoot(dependencies)
		Expect(bootRecord.CurrentBootMode).To(Equal("bios"))
		Expect(bootRecord.PxeInterface).To(Equal(""))
		fileInfoMock.AssertNotCalled(GinkgoT(), "IsDir")
	})

	It("no pxe+not dir", func() {
		fileInfoMock := MockFileInfo{}
		fileInfoMock.On("IsDir").Return(false).Once()
		dependencies.On("Stat", "/sys/firmware/efi").Return(&fileInfoMock, nil).Once()
		dependencies.On("ReadFile", "/proc/cmdline").Return([]byte(cmdlineNoPxe), nil)
		bootRecord := GetBoot(dependencies)
		Expect(bootRecord.CurrentBootMode).To(Equal("bios"))
		Expect(bootRecord.PxeInterface).To(Equal(""))
		fileInfoMock.AssertExpectations(GinkgoT())
	})

	It("pxe+dir", func() {
		fileInfoMock := MockFileInfo{}
		fileInfoMock.On("IsDir").Return(true).Once()
		dependencies.On("Stat", "/sys/firmware/efi").Return(&fileInfoMock, nil).Once()
		dependencies.On("ReadFile", "/proc/cmdline").Return([]byte(cmdlineWithPxe), nil)
		bootRecord := GetBoot(dependencies)
		Expect(bootRecord.CurrentBootMode).To(Equal("uefi"))
		Expect(bootRecord.PxeInterface).To(Equal("80:32:53:4f:cf:d6"))
		fileInfoMock.AssertExpectations(GinkgoT())
	})

	It("cmdline only", func() {
		fileInfoMock := MockFileInfo{}
		fileInfoMock.On("IsDir").Return(true).Once()
		dependencies.On("ReadFile", "/proc/cmdline").Return([]byte(cmdlines390x), nil)
		bootRecord := GetBoot(dependencies)
		Expect(bootRecord.CommandLine).To(Equal(cmdlines390x))
		fileInfoMock.AssertExpectations(GinkgoT())
	})
})
