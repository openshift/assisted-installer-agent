package inventory

import (
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"

	"github.com/openshift/assisted-service/models"
)

const (
	BytesMultiplier int64 = 1
	KbMultiplier          = BytesMultiplier << 10
	MbMultiplier          = KbMultiplier << 10
	GbMultiplier          = MbMultiplier << 10
	TbMultiplier          = GbMultiplier << 10
	EbMultiplier          = TbMultiplier << 10
	ZbMultiplier          = EbMultiplier << 10
)

var multiplierMap = map[string]int64{
	"bytes": BytesMultiplier,
	"kb":    KbMultiplier,
	"mb":    MbMultiplier,
	"gb":    GbMultiplier,
	"tb":    TbMultiplier,
	"eb":    EbMultiplier,
	"zb":    ZbMultiplier,
}

type memory struct {
	dependencies util.IDependencies
}

func newMemory(dependencies util.IDependencies) *memory {
	return &memory{dependencies: dependencies}
}

func (m *memory) getTotalPhysicalBytes() int64 {

	if physicalBytes := m.getDmidecodeTotalPhysicalBytes(); physicalBytes != 0 {
		return physicalBytes
	}

	// Try to use ghw to get total physical bytes, as dmidecode returns 0 on some platforms
	if physicalBytes := m.getGhwTotalPhysicalBytes(); physicalBytes != 0 {
		return physicalBytes
	}

	// Treat usable bytes as physical bytes, as ghw returns 0 on some platforms
	if usableBytes := m.getTotalUsabeBytes(); usableBytes != 0 {
		return usableBytes
	}

	return 0
}

func (m *memory) getGhwTotalPhysicalBytes() int64 {
	mem, err := m.dependencies.Memory()
	if err != nil {
		logrus.Errorf("Error getting memory info: %v", err)
		return 0
	}

	return mem.TotalPhysicalBytes
}

func (m *memory) getDmidecodeTotalPhysicalBytes() int64 {
	o, e, exitCode := m.dependencies.Execute("dmidecode", "-t", "17")
	if exitCode != 0 {
		logrus.Errorf("Could not run dmidecode: %s", e)
		return 0
	}
	r := regexp.MustCompile("^[ \t]*Size:[ \t]+([0-9]+)[ \t]+([a-zA-Z]+)[ \t]*$")
	var total int64
	for _, line := range strings.Split(o, "\n") {
		matches := r.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		value, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			logrus.Warnf("Could not convert memory: %s", err.Error())
			return 0
		}
		multiplier, ok := multiplierMap[strings.ToLower(matches[2])]
		if !ok {
			logrus.Warnf("Could not find multiplier for unit %s", matches[2])
			return 0
		}
		total += value * multiplier
	}
	return total
}

func (m *memory) getTotalUsabeBytes() int64 {
	b, err := m.dependencies.ReadFile("/proc/meminfo")
	if err != nil {
		logrus.WithError(err).Error("Read /proc/meminfo")
		return 0
	}
	exp := regexp.MustCompile("^[ \t]*MemTotal:[ \t]+([0-9]+)[ \t]+([a-zA-Z]+)")
	for _, line := range strings.Split(string(b), "\n") {
		matches := exp.FindStringSubmatch(line)
		if len(matches) == 3 {
			value, err := strconv.ParseInt(matches[1], 10, 64)
			if err != nil {
				logrus.WithError(err).Errorf("During conversion of %s", matches[2])
				return 0
			}
			multiplier, ok := multiplierMap[strings.ToLower(matches[2])]
			if !ok {
				logrus.Errorf("Could not find multiplier for unit %s", matches[2])
				return 0
			}
			return value * multiplier
		}
	}
	logrus.Error("Could not find MemTotal in /proc/meminfo")
	return 0
}

func (m *memory) getMemory() *models.Memory {
	ret := models.Memory{
		PhysicalBytes: m.getTotalPhysicalBytes(),
		UsableBytes:   m.getTotalUsabeBytes(),
	}
	return &ret
}

func GetMemory(dependencies util.IDependencies) *models.Memory {
	return newMemory(dependencies).getMemory()
}
