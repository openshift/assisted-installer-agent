package inventory

import (
	"github.com/filanov/bm-inventory/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	lshwOutput = `{
    "id": "ibm-p8-05-fsp.mgmt.pnr.lab.eng.rdu2.redhat.com",
    "class": "system",
    "claimed": true,
    "handle": "DMI:0012",
    "description": "Notebook",
    "product": "20NYS7K91V (LENOVO_MT_20NY_BU_Think_FM_ThinkPad T490s)",
    "vendor": "LENOVO",
    "version": "ThinkPad T490s",
    "serial": "PC1E81XA",
    "width": 64,
    "configuration": {
        "administrator_password": "disabled",
        "chassis": "notebook",
        "family": "ThinkPad T490s",
        "power-on_password": "disabled",
        "sku": "LENOVO_MT_20NY_BU_Think_FM_ThinkPad T490s",
        "uuid": "CC5D820B-6430-B211-A85C-968EC80B7FCF"
    },
    "capabilities": {
        "smbios-3.1.1": "SMBIOS version 3.1.1",
        "dmi-3.1.1": "DMI version 3.1.1",
        "smp": "Symmetric Multi-Processing",
        "vsyscall32": "32-bit processes"
    },
    "children": [
        {
            "id": "core",
            "class": "bus",
            "claimed": true,
            "handle": "DMI:0013",
            "description": "Motherboard",
            "product": "20NYS7K91V",
            "vendor": "LENOVO",
            "physid": "0",
            "version": "Not Defined",
            "serial": "L1HF02A02SD",
            "slot": "Not Available",
            "children": [
                {
                    "id": "battery",
                    "class": "power",
                    "claimed": true,
                    "handle": "DMI:002C",
                    "product": "02DL014",
                    "vendor": "SMP",
                    "physid": "1",
                    "slot": "Front",
                    "units": "mWh",
                    "capacity": 57020,
                    "configuration": {
                        "voltage": "11.5V"
                    }
                }
            ]
        }
    ]
}
`
	malformedLshwOutput = `{
    "id": "ibm-p8-05-fsp.mgmt.pnr.lab.eng.rdu2.redhat.com",
    "class": "system",
    "claimed": true,
    "handle": "DMI:0012",
    "description": "Notebook",
    "product": "20NYS7K91V (LENOVO_MT_20NY_BU_Think_FM_ThinkPad T490s)",
    "vendor": "LENOVO",
    "version": "ThinkPad T490s",
    "serial": "PC1E81XA",
    "width": 64,
    "configuration": yyy {
        "administrator_password": "disabled",
        "chassis": "notebook",
        "family": "ThinkPad T490s",
        "power-on_password": "disabled",
        "sku": "LENOVO_MT_20NY_BU_Think_FM_ThinkPad T490s",
        "uuid": "CC5D820B-6430-B211-A85C-968EC80B7FCF"
    },
    "capabilities": {
        "smbios-3.1.1": "SMBIOS version 3.1.1",
        "dmi-3.1.1": "DMI version 3.1.1",
        "smp": "Symmetric Multi-Processing",
        "vsyscall32": "32-bit processes"
    },
    "children": [
        {
            "id": "core",
            "class": "bus",
            "claimed": true,
            "handle": "DMI:0013",
            "description": "Motherboard",
            "product": "20NYS7K91V",
            "vendor": "LENOVO",
            "physid": "0",
            "version": "Not Defined",
            "serial": "L1HF02A02SD",
            "slot": "Not Available",
            "children": [
                {
                    "id": "battery",
                    "class": "power",
                    "claimed": true,
                    "handle": "DMI:002C",
                    "product": "02DL014",
                    "vendor": "SMP",
                    "physid": "1",
                    "slot": "Front",
                    "units": "mWh",
                    "capacity": 57020,
                    "configuration": {
                        "voltage": "11.5V"
                    }
                }
            ]
        }
    ]
}
`
)

var _ = Describe("System vendor test", func() {
	var dependencies *MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("Execute error", func() {
		dependencies.On("Execute", "lshw", "-quiet", "-json").Return(lshwOutput, "Execute error", -1)
		ret := GetVendor(dependencies)
		Expect(ret).To(Equal(&models.SystemVendor{}))
	})
	It("Json error", func() {
		dependencies.On("Execute", "lshw", "-quiet", "-json").Return(malformedLshwOutput, "", 0)
		ret := GetVendor(dependencies)
		Expect(ret).To(Equal(&models.SystemVendor{}))
	})
	It("lshw OK", func() {
		dependencies.On("Execute", "lshw", "-quiet", "-json").Return(lshwOutput, "", 0)
		ret := GetVendor(dependencies)
		Expect(ret).To(Equal(&models.SystemVendor{
			Manufacturer: "LENOVO",
			ProductName:  "20NYS7K91V (LENOVO_MT_20NY_BU_Think_FM_ThinkPad T490s)",
			SerialNumber: "PC1E81XA",
		}))
	})
})
