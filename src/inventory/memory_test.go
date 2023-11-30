package inventory

import (
	"fmt"

	"github.com/jaypipes/ghw"
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

var (
	mem1 = ghw.MemoryArea{
		TotalPhysicalBytes: 0,
		TotalUsableBytes:   0,
		SupportedPageSizes: nil,
		Modules:            nil,
	}

	mem2 = ghw.MemoryArea{
		TotalPhysicalBytes: 35184372088832,
		TotalUsableBytes:   34244109795328,
		SupportedPageSizes: nil,
		Modules:            nil,
	}
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
		dependencies.On("Execute", "dmidecode", "-t", "17").Return("", "dmidecode error", -1).Once()
		dependencies.On("ReadFile", "/proc/meminfo").Return(nil, fmt.Errorf("meminfo error")).Twice()
		dependencies.On("Memory").Return(&mem1, nil).Once()

		ret := GetMemory(dependencies)
		Expect(ret).To(Equal(&models.Memory{}))
		Expect(ret.PhysicalBytes).To(Equal(int64(0)))
		Expect(string(ret.PhysicalBytesMethod)).To(Equal(""))
		Expect(ret.UsableBytes).To(Equal(int64(0)))
	})
	It("dmidecode fallback to ghw", func() {
		dependencies.On("Execute", "dmidecode", "-t", "17").Return("", "Just an error", -1).Once()
		dependencies.On("ReadFile", "/proc/meminfo").Return([]byte(meminfoContentskB), nil).Once()
		dependencies.On("Memory").Return(&mem2, nil).Once()
		ret := GetMemory(dependencies)
		Expect(ret.PhysicalBytes).To(Equal(mem2.TotalPhysicalBytes))
		Expect(ret.PhysicalBytesMethod).To(Equal(models.MemoryMethodGhw))
	})
	It("ghw fallback to usable", func() {
		dependencies.On("Execute", "dmidecode", "-t", "17").Return("", "Just an error", -1).Once()
		dependencies.On("ReadFile", "/proc/meminfo").Return([]byte(meminfoContentskB), nil).Twice()
		dependencies.On("Memory").Return(&mem1, nil).Once()
		ret := GetMemory(dependencies)
		Expect(ret.PhysicalBytes).To(Equal(ret.UsableBytes))
		Expect(ret.PhysicalBytesMethod).To(Equal(models.MemoryMethodMeminfo))
	})

	It("Execute MB+read kB OK", func() {
		dependencies.On("Execute", "dmidecode", "-t", "17").Return(dmidecodeOutputMB, "", 0).Once()
		dependencies.On("ReadFile", "/proc/meminfo").Return([]byte(meminfoContentskB), nil).Once()
		ret := GetMemory(dependencies)
		Expect(ret).To(Equal(&models.Memory{
			PhysicalBytes:       34359738368,
			UsableBytes:         33441513472,
			PhysicalBytesMethod: models.MemoryMethodDmidecode,
		}))
	})
	It("Execute GB+read MB OK", func() {
		dependencies.On("Execute", "dmidecode", "-t", "17").Return(dmidecodeOutputGB, "", 0).Once()
		dependencies.On("ReadFile", "/proc/meminfo").Return([]byte(meminfoContentsMB), nil).Once()
		ret := GetMemory(dependencies)
		Expect(ret).To(Equal(&models.Memory{
			PhysicalBytes:       35184372088832,
			UsableBytes:         34244109795328,
			PhysicalBytesMethod: models.MemoryMethodDmidecode,
		}))
	})
})
