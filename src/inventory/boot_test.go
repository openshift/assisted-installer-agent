package inventory

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

type FileInfoMock struct {
	mock.Mock
}

func (f *FileInfoMock) Name() string {
	args := f.Called()
	return args.String(0)
}

func (f *FileInfoMock) Size() int64 {
	_ = f.Called()
	return 0
}

func (f *FileInfoMock) Mode() os.FileMode {
	_ = f.Called()
	return 0
}

func (f *FileInfoMock) ModTime() time.Time {
	_ = f.Called()
	return time.Time{}
}

func (f *FileInfoMock) IsDir() bool {
	args := f.Called()
	return args.Bool(0)
}
func (f *FileInfoMock) Sys() interface{} {
	_ = f.Called()
	return 0
}

const (
	cmdlineNoPxe   = `BOOT_IMAGE=(hd0,gpt2)/vmlinuz-5.5.17-200.fc31.x86_64 root=/dev/mapper/fedora_dhcp--0--223-root ro resume=/dev/mapper/fedora_dhcp--0--223-swap rd.lvm.lv=fedora_dhcp-0-223/root rd.luks.uuid=luks-47bb99f4-7573-42cf-bfcf-92aaa826fb9b rd.lvm.lv=fedora_dhcp-0-223/swap rhgb quiet psmouse.elantech_smbus=0 systemd.unified_cgroup_hierarchy=0`
	cmdlineWithPxe = `BOOT_IMAGE=(hd0,gpt2)/vmlinuz-5.5.17-200.fc31.x86_64 root=/dev/mapper/fedora_dhcp--0--223-root ro BOOTIF=80:32:53:4f:cf:d6 resume=/dev/mapper/fedora_dhcp--0--223-swap rd.lvm.lv=fedora_dhcp-0-223/root rd.luks.uuid=luks-47bb99f4-7573-42cf-bfcf-92aaa826fb9b rd.lvm.lv=fedora_dhcp-0-223/swap rhgb quiet psmouse.elantech_smbus=0 systemd.unified_cgroup_hierarchy=0`
)

var _ = Describe("boot", func() {

	var dependencies *MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("pxe+mode fail", func() {
		fileInfoMock := FileInfoMock{}
		fileInfoMock.On("IsDir").Return(true)
		dependencies.On("Stat", "/sys/firmware/efi").Return(&fileInfoMock, fmt.Errorf("Just error")).Once()
		dependencies.On("ReadFile", "/proc/cmdline").Return(nil, fmt.Errorf("Just another error"))
		bootRecord := GetBoot(dependencies)
		Expect(bootRecord.CurrentBootMode).To(Equal("bios"))
		Expect(bootRecord.PxeInterface).To(Equal(""))
		fileInfoMock.AssertNotCalled(GinkgoT(), "IsDir")
	})

	It("no pxe+not dir", func() {
		fileInfoMock := FileInfoMock{}
		fileInfoMock.On("IsDir").Return(false).Once()
		dependencies.On("Stat", "/sys/firmware/efi").Return(&fileInfoMock, nil).Once()
		dependencies.On("ReadFile", "/proc/cmdline").Return([]byte(cmdlineNoPxe), nil)
		bootRecord := GetBoot(dependencies)
		Expect(bootRecord.CurrentBootMode).To(Equal("bios"))
		Expect(bootRecord.PxeInterface).To(Equal(""))
		fileInfoMock.AssertExpectations(GinkgoT())
	})

	It("pxe+dir", func() {
		fileInfoMock := FileInfoMock{}
		fileInfoMock.On("IsDir").Return(true).Once()
		dependencies.On("Stat", "/sys/firmware/efi").Return(&fileInfoMock, nil).Once()
		dependencies.On("ReadFile", "/proc/cmdline").Return([]byte(cmdlineWithPxe), nil)
		bootRecord := GetBoot(dependencies)
		Expect(bootRecord.CurrentBootMode).To(Equal("uefi"))
		Expect(bootRecord.PxeInterface).To(Equal("80:32:53:4f:cf:d6"))
		fileInfoMock.AssertExpectations(GinkgoT())
	})
})
