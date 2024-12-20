package inventory

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

const (
	goodLscpuOutput = `{
   "lscpu": [
      {"field":"Architecture:", "data":"x86_64"},
      {"field":"CPU op-mode(s):", "data":"32-bit, 64-bit"},
      {"field":"Byte Order:", "data":"Little Endian"},
      {"field":"Address sizes:", "data":"39 bits physical, 48 bits virtual"},
      {"field":"CPU(s):", "data":"8"},
      {"field":"On-line CPU(s) list:", "data":"0-7"},
      {"field":"Thread(s) per core:", "data":"2"},
      {"field":"Core(s) per socket:", "data":"4"},
      {"field":"Socket(s):", "data":"1"},
      {"field":"NUMA node(s):", "data":"1"},
      {"field":"Vendor ID:", "data":"GenuineIntel"},
      {"field":"CPU family:", "data":"6"},
      {"field":"Model:", "data":"142"},
      {"field":"Model name:", "data":"Intel(R) Core(TM) i7-8665U CPU @ 1.90GHz"},
      {"field":"Stepping:", "data":"12"},
      {"field":"CPU MHz:", "data":"3593.210"},
      {"field":"CPU max MHz:", "data":"4800.0000"},
      {"field":"CPU min MHz:", "data":"400.0000"},
      {"field":"BogoMIPS:", "data":"4199.88"},
      {"field":"Virtualization:", "data":"VT-x"},
      {"field":"L1d cache:", "data":"128 KiB"},
      {"field":"L1i cache:", "data":"128 KiB"},
      {"field":"L2 cache:", "data":"1 MiB"},
      {"field":"L3 cache:", "data":"8 MiB"},
      {"field":"NUMA node0 CPU(s):", "data":"0-7"},
      {"field":"Vulnerability Itlb multihit:", "data":"KVM: Mitigation: Split huge pages"},
      {"field":"Vulnerability L1tf:", "data":"Not affected"},
      {"field":"Vulnerability Mds:", "data":"Not affected"},
      {"field":"Vulnerability Meltdown:", "data":"Not affected"},
      {"field":"Vulnerability Spec store bypass:", "data":"Mitigation; Speculative Store Bypass disabled via prctl and seccomp"},
      {"field":"Vulnerability Spectre v1:", "data":"Mitigation; usercopy/swapgs barriers and __user pointer sanitization"},
      {"field":"Vulnerability Spectre v2:", "data":"Mitigation; Enhanced IBRS, IBPB conditional, RSB filling"},
      {"field":"Vulnerability Tsx async abort:", "data":"Mitigation; TSX disabled"},
      {"field":"Flags:", "data":"fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush dts acpi mmx fxsr sse sse2 ss ht tm pbe syscall nx pdpe1gb rdtscp lm constant_tsc art arch_perfmon pebs bts rep_good nopl xtopology nonstop_tsc cpuid aperfmperf pni pclmulqdq dtes64 monitor ds_cpl vmx smx est tm2 ssse3 sdbg fma cx16 xtpr pdcm pcid sse4_1 sse4_2 x2apic movbe popcnt tsc_deadline_timer aes xsave avx f16c rdrand lahf_lm abm 3dnowprefetch cpuid_fault epb invpcid_single ssbd ibrs ibpb stibp ibrs_enhanced tpr_shadow vnmi flexpriority ept vpid ept_ad fsgsbase tsc_adjust bmi1 avx2 smep bmi2 erms invpcid mpx rdseed adx smap clflushopt intel_pt xsaveopt xsavec xgetbv1 xsaves dtherm ida arat pln pts hwp hwp_notify hwp_act_window hwp_epp md_clear flush_l1d arch_capabilities"}
   ]
}`

	malformedLscpuOutput = `{
   "lscpu": "Hello world"[
      {"field":"Architecture:", "data":"x86_64"},
      {"field":"CPU op-mode(s):", "data":"32-bit, 64-bit"},
      {"field":"Byte Order:", "data":"Little Endian"},
      {"field":"Address sizes:", "data":"39 bits physical, 48 bits virtual"},
      {"field":"CPU(s):", "data":"8"},
      {"field":"On-line CPU(s) list:", "data":"0-7"},
      {"field":"Thread(s) per core:", "data":"2"},
      {"field":"Core(s) per socket:", "data":"4"},
      {"field":"Socket(s):", "data":"1"},
      {"field":"NUMA node(s):", "data":"1"},
      {"field":"Vendor ID:", "data":"GenuineIntel"},
      {"field":"CPU family:", "data":"6"},
      {"field":"Model:", "data":"142"},
      {"field":"Model name:", "data":"Intel(R) Core(TM) i7-8665U CPU @ 1.90GHz"},
      {"field":"Stepping:", "data":"12"},
      {"field":"CPU MHz:", "data":"3593.210"},
      {"field":"CPU max MHz:", "data":"4800.0000"},
      {"field":"CPU min MHz:", "data":"400.0000"},
      {"field":"BogoMIPS:", "data":"4199.88"},
      {"field":"Virtualization:", "data":"VT-x"},
      {"field":"L1d cache:", "data":"128 KiB"},
      {"field":"L1i cache:", "data":"128 KiB"},
      {"field":"L2 cache:", "data":"1 MiB"},
      {"field":"L3 cache:", "data":"8 MiB"},
      {"field":"NUMA node0 CPU(s):", "data":"0-7"},
      {"field":"Vulnerability Itlb multihit:", "data":"KVM: Mitigation: Split huge pages"},
      {"field":"Vulnerability L1tf:", "data":"Not affected"},
      {"field":"Vulnerability Mds:", "data":"Not affected"},
      {"field":"Vulnerability Meltdown:", "data":"Not affected"},
      {"field":"Vulnerability Spec store bypass:", "data":"Mitigation; Speculative Store Bypass disabled via prctl and seccomp"},
      {"field":"Vulnerability Spectre v1:", "data":"Mitigation; usercopy/swapgs barriers and __user pointer sanitization"},
      {"field":"Vulnerability Spectre v2:", "data":"Mitigation; Enhanced IBRS, IBPB conditional, RSB filling"},
      {"field":"Vulnerability Tsx async abort:", "data":"Mitigation; TSX disabled"},
      {"field":"Flags:", "data":"fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush dts acpi mmx fxsr sse sse2 ss ht tm pbe syscall nx pdpe1gb rdtscp lm constant_tsc art arch_perfmon pebs bts rep_good nopl xtopology nonstop_tsc cpuid aperfmperf pni pclmulqdq dtes64 monitor ds_cpl vmx smx est tm2 ssse3 sdbg fma cx16 xtpr pdcm pcid sse4_1 sse4_2 x2apic movbe popcnt tsc_deadline_timer aes xsave avx f16c rdrand lahf_lm abm 3dnowprefetch cpuid_fault epb invpcid_single ssbd ibrs ibpb stibp ibrs_enhanced tpr_shadow vnmi flexpriority ept vpid ept_ad fsgsbase tsc_adjust bmi1 avx2 smep bmi2 erms invpcid mpx rdseed adx smap clflushopt intel_pt xsaveopt xsavec xgetbv1 xsaves dtherm ida arat pln pts hwp hwp_notify hwp_act_window hwp_epp md_clear flush_l1d arch_capabilities"}
   ]
}`

	s390xLscpuOutput = `{
   "lscpu": [
      {"field":"Architecture:", "data":"s390x"},
      {"field":"CPU op-mode(s):", "data":"32-bit, 64-bit"},
      {"field":"Byte Order:", "data":"Big Endian"},
      {"field":"CPU(s):", "data":"4"},
      {"field":"On-line CPU(s) list:", "data":"0-3"},
      {"field":"Vendor ID:", "data":"IBM/S390"},
      {"field":"Machine type:", "data":"3931"},
      {"field":"Thread(s) per core:", "data":"1"},
      {"field":"Core(s) per socket:", "data":"1"},
      {"field":"Socket(s):", "data":"1"},
      {"field":"Book(s) per drawer:", "data":"1"},
      {"field":"Drawer(s):", "data":"4"},
      {"field":"CPU dynamic MHz:", "data":"5200"},
      {"field":"CPU static MHz:", "data":"5200"},
      {"field":"BogoMIPS:", "data":"26315.00"},
      {"field":"Dispatching mode:", "data":"horizontal"},
      {"field":"Flags:", "data":"esan3 zarch stfle msa ldisp eimm dfp edat etf3eh highgprs te vx vxd vxe gs vxe2 vxp sort dflt vxp2 nnpa sie"},
      {"field":"Hypervisor:", "data":"z/VM 7.3.0"},
      {"field":"Hypervisor vendor:", "data":"IBM"},
      {"field":"Virtualization type:", "data":"full"},
      {"field":"L1d:", "data":"512 KiB (4 instances)"},
      {"field":"L1i:", "data":"512 KiB (4 instances)"},
      {"field":"L2:", "data":"32 MiB (1 instance)"},
      {"field":"L3:", "data":"256 MiB"},
      {"field":"NUMA node(s):", "data":"1"},
      {"field":"NUMA node0 CPU(s):", "data":"0-3"},
      {"field":"Gather data sampling:", "data":"Not affected"},
      {"field":"Itlb multihit:", "data":"Not affected"},
      {"field":"L1tf:", "data":"Not affected"},
      {"field":"Mds:", "data":"Not affected"},
      {"field":"Meltdown:", "data":"Not affected"},
      {"field":"Mmio stale data:", "data":"Not affected"},
      {"field":"Retbleed:", "data":"Not affected"},
      {"field":"Spec store bypass:", "data":"Not affected"},
      {"field":"Spectre v1:", "data":"Mitigation; __user pointer sanitization"},
      {"field":"Spectre v2:", "data":"Mitigation; etokens"},
      {"field":"Srbds:", "data":"Not affected"},
      {"field":"Tsx async abort:", "data":"Not affected"}
   ]
}`
)

var _ = Describe("CPU test", func() {

	var dependencies *util.MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("Execute error", func() {
		dependencies.On("Execute", "lscpu", "-J").Return(goodLscpuOutput, "Execute error", -1).Once()
		ret := GetCPU(dependencies)
		Expect(ret).To(Equal(&models.CPU{}))
	})
	It("Json error", func() {
		dependencies.On("Execute", "lscpu", "-J").Return(malformedLscpuOutput, "", 0).Once()
		ret := GetCPU(dependencies)
		Expect(ret).To(Equal(&models.CPU{}))
	})
	It("lscpu OK", func() {
		dependencies.On("Execute", "lscpu", "-J").Return(goodLscpuOutput, "", 0).Once()
		ret := GetCPU(dependencies)
		expected := models.CPU{
			Architecture: "x86_64",
			Count:        8,
			Flags: []string{"fpu", "vme", "de", "pse", "tsc", "msr", "pae", "mce", "cx8", "apic", "sep", "mtrr", "pge",
				"mca", "cmov", "pat", "pse36", "clflush", "dts", "acpi", "mmx", "fxsr", "sse", "sse2", "ss", "ht", "tm",
				"pbe", "syscall", "nx", "pdpe1gb", "rdtscp", "lm", "constant_tsc", "art", "arch_perfmon", "pebs", "bts",
				"rep_good", "nopl", "xtopology", "nonstop_tsc", "cpuid", "aperfmperf", "pni", "pclmulqdq", "dtes64", "monitor",
				"ds_cpl", "vmx", "smx", "est", "tm2", "ssse3", "sdbg", "fma", "cx16", "xtpr", "pdcm", "pcid", "sse4_1", "sse4_2",
				"x2apic", "movbe", "popcnt", "tsc_deadline_timer", "aes", "xsave", "avx", "f16c", "rdrand", "lahf_lm", "abm",
				"3dnowprefetch", "cpuid_fault", "epb", "invpcid_single", "ssbd", "ibrs", "ibpb", "stibp", "ibrs_enhanced", "tpr_shadow",
				"vnmi", "flexpriority", "ept", "vpid", "ept_ad", "fsgsbase", "tsc_adjust", "bmi1", "avx2", "smep", "bmi2", "erms", "invpcid",
				"mpx", "rdseed", "adx", "smap", "clflushopt", "intel_pt", "xsaveopt", "xsavec", "xgetbv1", "xsaves", "dtherm", "ida",
				"arat", "pln", "pts", "hwp", "hwp_notify", "hwp_act_window", "hwp_epp", "md_clear", "flush_l1d", "arch_capabilities"},
			Frequency: 4800,
			ModelName: "Intel(R) Core(TM) i7-8665U CPU @ 1.90GHz",
		}
		Expect(ret).To(Equal(&expected))
	})
	It("s390x OK", func() {
		dependencies.On("Execute", "lscpu", "-J").Return(s390xLscpuOutput, "", 0).Once()
		ret := GetCPU(dependencies)
		expected := models.CPU{
			Architecture: "s390x",
			Count:        4,
			Flags: []string{"esan3", "zarch", "stfle", "msa", "ldisp", "eimm", "dfp", "edat", "etf3eh", "highgprs", "te", "vx",
				"vxd", "vxe", "gs", "vxe2", "vxp", "sort", "dflt", "vxp2", "nnpa", "sie"},
			Frequency: 5200,
			ModelName: "3931",
		}
		Expect(ret).To(Equal(&expected))
	})
})
