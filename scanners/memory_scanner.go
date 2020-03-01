package scanners

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strconv"
	"strings"
)

const (
	TOTAL_LABLE = "total"
	USED_LABLE = "used"
	FREE_LABLE = "free"
	SHARED_LABLE = "shared"
	BUFF_CACHE_LABLE = "buff/cache"
	AVAILABLE_LABLE = "available"
)

type MemoryInfo struct {
	Name  string
	Total uint64
	Used  uint64
	Free  uint64
	Shared uint64
	BuffCached uint64
	Available uint64
}


func readHeader(header string) map[int] string {
	ret := make(map[int]string)
	for token, start := nextToken(header, 0) ; start < len(header) ; token, start = nextToken(header, start + len(token)) {
		ret[start + len(token) -1] = token
	}
	return ret
}

func ReadMemory() []MemoryInfo {
	cmd := exec.Command("free")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running free: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	if len(lines) == 0 {
		log.Warnf("ReadMemoey: Missing lines")
		return nil
	}
	ret := make([]MemoryInfo, 0)
	headerMap := readHeader(lines[0])
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		minfo := MemoryInfo{}
		for token, start := nextToken(line, 0) ; start < len(line) ; token, start = nextToken(line, start + len(token)) {
			switch start {
			case 0:
				minfo.Name = token[:len(token) -1]
			default:
				lable, ok := headerMap[start + len(token) - 1]
				if !ok {
					continue
				}
				value, _  := strconv.ParseUint(token, 10, 64)
				switch lable {
				case TOTAL_LABLE:
					minfo.Total = value
				case USED_LABLE:
					minfo.Used = value
				case FREE_LABLE:
					minfo.Free = value
				case SHARED_LABLE:
					minfo.Shared = value
				case BUFF_CACHE_LABLE:
					minfo.BuffCached = value
				case AVAILABLE_LABLE:
					minfo.Available = value
				}
			}
		}
		ret = append(ret , minfo)
	}
	return ret
}
