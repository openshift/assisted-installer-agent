package scanners

import (
	"github.com/filanov/bm-inventory/models"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strconv"
	"strings"
)

const (
	NAME_LABLE = "NAME"
	MAJ_MIN_LABLE = "MAJ:MIN"
	RM_LABLE = "RM"
	SIZE_LABLE = "SIZE"
	RO_LABLE = "RO"
	TYPE_LABLE = "TYPE"
	MOUNTPOINT_LABLE = "MOUNTPOINT"
	FSTYPE_LABLE = "FSTYPE"
	LEFT_ALIGHNMENT = "left"
	RIGHT_ALIGNMENT = "right"
	COLON_ALIGNMENT = "colon"
)



var lables = map[string] string {
	NAME_LABLE: LEFT_ALIGHNMENT,
	MAJ_MIN_LABLE: COLON_ALIGNMENT,
	RM_LABLE: RIGHT_ALIGNMENT,
	SIZE_LABLE: RIGHT_ALIGNMENT,
	RO_LABLE: RIGHT_ALIGNMENT,
	TYPE_LABLE: LEFT_ALIGHNMENT,
	MOUNTPOINT_LABLE: LEFT_ALIGHNMENT,
	FSTYPE_LABLE: LEFT_ALIGHNMENT,
}

func mapHeader(header string) map[int]string {
	ret := make(map[int]string)
	for l,alighment := range lables {
		index := strings.Index(header, l)
		if index == -1 {
			log.Warnf("No index found for %s", l)
			continue
		}
		switch alighment {
		case LEFT_ALIGHNMENT:
			ret[index] = l
		case RIGHT_ALIGNMENT:
			ret[index + len(l) - 1] = l
		case COLON_ALIGNMENT:
			colonIndex := strings.Index(l, ":")
			ret[index + colonIndex] = l
		}
	}
	return ret
}

func nextToken(line string, start int) (token string, begin int) {
	ret := ""
	for ; start < len(line) && line[start] == ' ' ; start++ {
	}
	begin = start
	for ; start < len(line) && line[start] != ' '; start++ {
		ret = ret + string(line[start])
	}
	return ret, begin
}

func ReadBlockDevices() [] *models.BlockDevice {
	cmd := exec.Command("lsblk", "-lab",  "-o",  "+FSTYPE")
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
	ret := make([] *models.BlockDevice, 0)
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		binfo := models.BlockDevice{}
		for token, start := nextToken(line, 0) ; start < len(line) ; token, start = nextToken(line, start + len(token)) {
			lable, ok := headersMap[start]
			if !ok {
				lable, ok = headersMap[start + len(token) -1]
			}
			if !ok {
				colonIndex := strings.Index(token, ":")
				lable, ok = headersMap[start + colonIndex]
			}
			if !ok {
				continue
			}
			switch lable {
			case NAME_LABLE:
				binfo.Name = token
			case FSTYPE_LABLE:
				binfo.Fstype = token
			case TYPE_LABLE:
				binfo.DeviceType = token
			case MOUNTPOINT_LABLE:
				binfo.Mountpoint = token
			case SIZE_LABLE:
				binfo.Size, _ = strconv.ParseInt(token, 10, 64)
			case RO_LABLE:
				binfo.ReadOnly = token != "0"
			case RM_LABLE:
				binfo.RemovableDevice, _ = strconv.ParseInt(token, 10, 64)
			case MAJ_MIN_LABLE:
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