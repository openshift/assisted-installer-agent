package scanners

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/filanov/bm-inventory/models"
	log "github.com/sirupsen/logrus"
)

const (
	TOTAL_LABEL      = "total"
	USED_LABEL       = "used"
	FREE_LABEL       = "free"
	SHARED_LABEL     = "shared"
	BUFF_CACHE_LABEL = "buff/cache"
	AVAILABLE_LABEL  = "available"
)

type MemoryInfo struct {
	Start int
	End   int
	Label string
}

func nextMemHeaderLabel(line string, start int) (label string, begin int, end int) {
	label = ""
	end = start
	for ; end < len(line) && line[end] == ' '; end++ {
	}
	for ; end < len(line) && line[end] != ' '; end++ {
		label = label + string(line[end])
	}
	return label, start, end
}

func readHeader(header string) []*MemoryInfo {
	ret := make([]*MemoryInfo, 0)
	for token, start, end := nextMemHeaderLabel(header, 0); start < len(header); token, start, end = nextMemHeaderLabel(header, end) {
		ret = append(ret, &MemoryInfo{
			Start: start,
			End:   end,
			Label: token,
		})
	}
	return ret
}

func max(x, y int) int {
	if x > y {
		return x
	} else {
		return y
	}
}

func ReadMemory() []*models.MemoryDetails {
	cmd := exec.Command("free", "-b")
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
	ret := make([]*models.MemoryDetails, 0)
	headerLabels := readHeader(lines[0])
	for _, line := range lines[1:] {
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}
		minfo := &models.MemoryDetails{}
		name, _ := nextToken(line, 0)
		minfo.Name = name[:len(name)-1]
		minStart := len(name)
		for _, m := range headerLabels {
			if m.End > len(line) {
				continue
			}
			token := strings.TrimSpace(line[max(minStart, m.Start):m.End])
			if token == "" {
				continue
			}
			value, _ := strconv.ParseInt(token, 10, 64)
			switch m.Label {
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
		ret = append(ret, minfo)
	}
	return ret
}
