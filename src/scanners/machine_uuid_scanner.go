package scanners

import (
	"crypto/md5"
	"fmt"
	"os"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/go-openapi/strfmt"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/util"
	log "github.com/sirupsen/logrus"
)

const (
	DefaultUUID = "00000000-0000-0000-0000-000000000000"
)

var unknownSerialCases = []string{"", util.UNKNOWN, "None", "Unspecified Base Board Serial Number"}

func disableGHWWarnings() {
	err := os.Setenv("GHW_DISABLE_WARNINGS", "1")
	if err != nil {
		log.WithError(err).Warn("Disable ghw warnings")
	}
}

//go:generate mockery -name SerialDiscovery -inpkg
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

	if value == "" || value == util.UNKNOWN {
		log.Warnf("Could not get system UUID.  Using default UUID %s", DefaultUUID)
		value = DefaultUUID
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
	if funk.Contains(unknownSerialCases, basedboard.SerialNumber) {
		return nil
	}
	return md5GenerateUUID(basedboard.SerialNumber)
}

func ReadId(d SerialDiscovery) *strfmt.UUID {
	ir := &idReader{serialDiscovery: d}
	ret := ir.readMotherboardSerial()
	if ret == nil {
		log.Warn("No valid motherboard serial, using system UUID instead")
		ret = ir.readSystemUUID()
	}
	return ret
}
