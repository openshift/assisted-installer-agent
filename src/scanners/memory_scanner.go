package scanners

import (
	"github.com/filanov/bm-inventory/models"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strconv"
	"strings"
)

const (
	TOTAL_LABEL = "total"
	USED_LABEL = "used"
	FREE_LABEL = "free"
	SHARED_LABEL = "shared"
	BUFF_CACHE_LABEL = "buff/cache"
	AVAILABLE_LABEL = "available"
)

func readHeader(header string) map[int] string {
	ret := make(map[int]string)
	for token, start := nextToken(header, 0) ; start < len(header) ; token, start = nextToken(header, start + len(token)) {
		ret[start + len(token) -1] = token
	}
	return ret
}

func ReadMemory() []*models.Memory {
	cmd := exec.Command("free")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running free: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	if len(lines) == 0 {
		log.Warnf("ReadMemory: Missing lines")
		return nil
	}
	ret := make([]*models.Memory, 0)
	headerMap := readHeader(lines[0])
	for _, line := range lines[1:] {
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}
		minfo := &models.Memory{}
		for token, start := nextToken(line, 0) ; start < len(line) ; token, start = nextToken(line, start + len(token)) {
			switch start {
			case 0:
				minfo.Name = token[:len(token) -1]
			default:
				label, ok := headerMap[start + len(token) - 1]
				if !ok {
					continue
				}
				value, _  := strconv.ParseInt(token, 10, 64)
				switch label {
				case TOTAL_LABEL:
					minfo.Total = value
				case USED_LABEL:
					minfo.Used = value
				case FREE_LABEL:
					minfo.Free = value
				case SHARED_LABEL:
					minfo.Shared = value
				case BUFF_CACHE_LABEL:
					minfo.BuffCached = value
				case AVAILABLE_LABEL:
					minfo.Available = value
				}
			}
		}
		ret = append(ret , minfo)
	}
	return ret
}
