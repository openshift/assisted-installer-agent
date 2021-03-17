package inventory

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

const (
	dmidecodeOutputMB = `# dmidecode 3.2
Getting SMBIOS data from sysfs.
SMBIOS 3.1.1 present.

Handle 0x0003, DMI type 17, 40 bytes
Memory Device
	Array Handle: 0x0002
	Error Information Handle: Not Provided
	Total Width: 64 bits
	Data Width: 64 bits
	Size: 16384 MB
	Form Factor: SODIMM
	Set: None
	Locator: ChannelA-DIMM0
	Bank Locator: BANK 0

Handle 0x0004, DMI type 17, 40 bytes
Memory Device
	Array Handle: 0x0002
	Error Information Handle: Not Provided
	Total Width: 64 bits
	Data Width: 64 bits
	Size: 16384 MB
	Form Factor: SODIMM
	Set: None
	Locator: ChannelB-DIMM0
	Bank Locator: BANK 2
	Type: DDR4
`
	dmidecodeOutputGB = `# dmidecode 3.2
Getting SMBIOS data from sysfs.
SMBIOS 3.1.1 present.

Handle 0x0003, DMI type 17, 40 bytes
Memory Device
	Array Handle: 0x0002
	Error Information Handle: Not Provided
	Total Width: 64 bits
	Data Width: 64 bits
	Size: 16384 GB
	Form Factor: SODIMM
	Set: None
	Locator: ChannelA-DIMM0
	Bank Locator: BANK 0

Handle 0x0004, DMI type 17, 40 bytes
Memory Device
	Array Handle: 0x0002
	Error Information Handle: Not Provided
	Total Width: 64 bits
	Data Width: 64 bits
	Size: 16384 GB
	Form Factor: SODIMM
	Set: None
	Locator: ChannelB-DIMM0
	Bank Locator: BANK 2
`
	meminfoContentskB = `MemTotal:       32657728 kB
MemFree:         7779692 kB
MemAvailable:   20752724 kB
Buffers:         1374624 kB
`
	meminfoContentsMB = `MemTotal:       32657728 MB
MemFree:         7779692 kB
MemAvailable:   20752724 kB
Buffers:         1374624 kB
Cached:         12556080 kB
`
)

var _ = Describe("Memory test", func() {
	var dependencies *util.MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("Execute+read error", func() {
		dependencies.On("Execute", "dmidecode", "-t", "17").Return("", "Just an error", -1).Once()
		dependencies.On("ReadFile", "/proc/meminfo").Return(nil, fmt.Errorf("Another error")).Once()
		ret := GetMemory(dependencies)
		Expect(ret).To(Equal(&models.Memory{}))
	})
	It("Execute MB+read kB OK", func() {
		dependencies.On("Execute", "dmidecode", "-t", "17").Return(dmidecodeOutputMB, "", 0).Once()
		dependencies.On("ReadFile", "/proc/meminfo").Return([]byte(meminfoContentskB), nil).Once()
		ret := GetMemory(dependencies)
		Expect(ret).To(Equal(&models.Memory{
			PhysicalBytes: 34359738368,
			UsableBytes:   33441513472,
		}))
	})
	It("Execute GB+read MB OK", func() {
		dependencies.On("Execute", "dmidecode", "-t", "17").Return(dmidecodeOutputGB, "", 0).Once()
		dependencies.On("ReadFile", "/proc/meminfo").Return([]byte(meminfoContentsMB), nil).Once()
		ret := GetMemory(dependencies)
		Expect(ret).To(Equal(&models.Memory{
			PhysicalBytes: 35184372088832,
			UsableBytes:   34244109795328,
		}))
	})
})
