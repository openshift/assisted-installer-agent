package scanners

import (
	"crypto/md5"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/go-openapi/strfmt"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/util"
	"github.com/openshift/assisted-installer-agent/src/inventory"
	agent_utils "github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

const (
	SerialDefaultString              = "default string"
	SerialUnspecifiedBaseBoardString = "unspecified base board serial number" // BF cards
	SerialUnspecifiedSystemString    = "unspecified system serial number"     // BF cards
	SerialNotSpecified               = "not specified"                        // Linode
	ZeroesUUID                       = "00000000-0000-0000-0000-000000000000"
	KaloomUUID                       = "03000200-0400-0500-0006-000700080009" // All hosts of this type have the same UUID
)

var (
	FailureUUID = strfmt.UUID("deaddead-dead-dead-dead-deaddeaddead")
)

var unknownSerialCases = []string{"", util.UNKNOWN, "none",
	SerialUnspecifiedBaseBoardString, SerialUnspecifiedSystemString,
	SerialDefaultString, SerialNotSpecified}
var unknownUuidCases = []string{"", util.UNKNOWN, ZeroesUUID, KaloomUUID}

func disableGHWWarnings() {
	err := os.Setenv("GHW_DISABLE_WARNINGS", "1")
	if err != nil {
		log.WithError(err).Warn("Disable ghw warnings")
	}
}

//go:generate mockery --name SerialDiscovery --inpackage
type SerialDiscovery interface {
	Product(opts ...*ghw.WithOption) (*ghw.ProductInfo, error)
	Baseboard(opts ...*ghw.WithOption) (*ghw.BaseboardInfo, error)
}

type GHWSerialDiscovery struct{}

func NewGHWSerialDiscovery() *GHWSerialDiscovery {
	disableGHWWarnings()
	return &GHWSerialDiscovery{}
}

func (g *GHWSerialDiscovery) Product(opts ...*ghw.WithOption) (*ghw.ProductInfo, error) {
	return ghw.Product(opts...)
}

func (g *GHWSerialDiscovery) Baseboard(opts ...*ghw.WithOption) (*ghw.BaseboardInfo, error) {
	return ghw.Baseboard(opts...)
}

func md5GenerateUUID(str string) *strfmt.UUID {
	md5Str := fmt.Sprintf("%x", md5.Sum([]byte(str)))
	uuidStr := strfmt.UUID(md5Str[0:8] + "-" + md5Str[8:12] + "-" + md5Str[12:16] + "-" + md5Str[16:20] + "-" + md5Str[20:])
	return &uuidStr
}

type idReader struct {
	serialDiscovery SerialDiscovery
}

func (ir *idReader) readSystemUUID() *strfmt.UUID {
	product, err := ir.serialDiscovery.Product()
	var value string
	if err != nil {
		log.Warnf("Could not find system UUID: %s", err.Error())
	} else {
		value = product.UUID
	}

	if !strfmt.IsUUID(value) || funk.Contains(unknownUuidCases, strings.ToLower(value)) {
		log.Warnf("Could not get system UUID. Got %s", value)
		return nil
	}

	ret := strfmt.UUID(strings.ToLower(value))
	return &ret
}

func (ir *idReader) readMotherboardSerial() *strfmt.UUID {
	basedboard, err := ir.serialDiscovery.Baseboard()
	if err != nil {
		log.WithError(err).Warningf("Failed to get motherboard serial")
		return nil
	}
	log.Infof("Motherboard serial number is %s", basedboard.SerialNumber)
	// serial can be unknown/unspecified or any other not serial case, we want to return nil
	if funk.Contains(unknownSerialCases, strings.ToLower(basedboard.SerialNumber)) {
		return nil
	}
	return md5GenerateUUID(basedboard.SerialNumber)
}

func uuidFromNetworkInterfaces(interfaces []*models.Interface) *strfmt.UUID {
	// remove interfaces with no mac address
	interfaces = funk.Filter(interfaces, func(iface interface{}) bool {
		return iface.(*models.Interface).MacAddress != ""
	}).([]*models.Interface)

	if len(interfaces) == 0 {
		return nil
	}

	// sort by mac
	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].MacAddress < interfaces[j].MacAddress
	})

	iface := interfaces[0]

	log.Infof("Using %s mac from interface %s to provide node-uuid", iface.MacAddress, iface.Name)
	return md5GenerateUUID(iface.MacAddress)
}

func ReadId(d SerialDiscovery, dependencies agent_utils.IDependencies) *strfmt.UUID {
	idReader := &idReader{serialDiscovery: d}

	motherboardSerialUUID := idReader.readMotherboardSerial()
	if motherboardSerialUUID != nil {
		return motherboardSerialUUID
	}

	log.Warn("No valid motherboard serial, using system UUID instead")
	systemUUID := idReader.readSystemUUID()
	if systemUUID != nil {
		return systemUUID
	}

	log.Warn("No valid system UUID, moving to network interfaces mac based UUID")
	interfacesUUID := uuidFromNetworkInterfaces(inventory.GetInterfaces(dependencies))
	if interfacesUUID != nil {
		return interfacesUUID
	}

	return &FailureUUID
}
