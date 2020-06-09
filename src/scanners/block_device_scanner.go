package scanners

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/filanov/bm-inventory/models"
	log "github.com/sirupsen/logrus"
)

const (
	NAME_LABEL       = "NAME"
	MAJ_MIN_LABEL    = "MAJ:MIN"
	RM_LABEL         = "RM"
	SIZE_LABEL       = "SIZE"
	RO_LABEL         = "RO"
	TYPE_LABEL       = "TYPE"
	MOUNTPOINT_LABEL = "MOUNTPOINT"
	FSTYPE_LABEL     = "FSTYPE"
	LEFT_ALIGHNMENT  = "left"
	RIGHT_ALIGNMENT  = "right"
	COLON_ALIGNMENT  = "colon"
)

var lables = map[string]string{
	NAME_LABEL:       LEFT_ALIGHNMENT,
	MAJ_MIN_LABEL:    COLON_ALIGNMENT,
	RM_LABEL:         RIGHT_ALIGNMENT,
	SIZE_LABEL:       RIGHT_ALIGNMENT,
	RO_LABEL:         RIGHT_ALIGNMENT,
	TYPE_LABEL:       LEFT_ALIGHNMENT,
	MOUNTPOINT_LABEL: LEFT_ALIGHNMENT,
	FSTYPE_LABEL:     LEFT_ALIGHNMENT,
}

func mapHeader(header string) map[int]string {
	ret := make(map[int]string)
	for l, alighment := range lables {
		index := strings.Index(header, l)
		if index == -1 {
			log.Warnf("No index found for %s", l)
			continue
		}
		switch alighment {
		case LEFT_ALIGHNMENT:
			ret[index] = l
		case RIGHT_ALIGNMENT:
			ret[index+len(l)-1] = l
		case COLON_ALIGNMENT:
			colonIndex := strings.Index(l, ":")
			ret[index+colonIndex] = l
		}
	}
	return ret
}

func nextToken(line string, start int) (token string, begin int) {
	ret := ""
	for ; start < len(line) && line[start] == ' '; start++ {
	}
	begin = start
	for ; start < len(line) && line[start] != ' '; start++ {
		ret = ret + string(line[start])
	}
	return ret, begin
}

func ReadBlockDevices() []*models.BlockDevice {
	cmd := exec.Command("lsblk", "-lab", "-o", "+FSTYPE")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running lsblk: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	if len(lines) == 0 {
		log.Warnf("No header found for lsblk")
		return nil
	}
	headersMap := mapHeader(lines[0])
	ret := make([]*models.BlockDevice, 0)
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		binfo := models.BlockDevice{}
		for token, start := nextToken(line, 0); start < len(line); token, start = nextToken(line, start+len(token)) {
			label, ok := headersMap[start]
			if !ok {
				label, ok = headersMap[start+len(token)-1]
			}
			if !ok {
				colonIndex := strings.Index(token, ":")
				label, ok = headersMap[start+colonIndex]
			}
			if !ok {
				continue
			}
			switch label {
			case NAME_LABEL:
				binfo.Name = token
			case FSTYPE_LABEL:
				binfo.Fstype = token
			case TYPE_LABEL:
				binfo.DeviceType = token
			case MOUNTPOINT_LABEL:
				binfo.Mountpoint = token
			case SIZE_LABEL:
				binfo.Size, _ = strconv.ParseInt(token, 10, 64)
			case RO_LABEL:
				binfo.ReadOnly = token != "0"
			case RM_LABEL:
				binfo.RemovableDevice, _ = strconv.ParseInt(token, 10, 64)
			case MAJ_MIN_LABEL:
				majMinSplit := strings.Split(token, ":")
				if len(majMinSplit) == 2 {
					binfo.MajorDeviceNumber, _ = strconv.ParseInt(majMinSplit[0], 10, 64)
					binfo.MinorDeviceNumber, _ = strconv.ParseInt(majMinSplit[1], 10, 64)
				}
			}
		}
		ret = append(ret, &binfo)
	}
	return ret
}
