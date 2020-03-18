package scanners

import (
	"crypto/md5"
	"fmt"
	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strings"
)

func getDmiValue(keyword string) string {
	cmd := exec.Command("dmidecode", "-s", keyword)
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running dmidecode on keyword %s: %s", keyword, err.Error())
		return ""
	}
	return strings.TrimSpace(string(bytes))
}

func md5GenerateUUID(str string) *strfmt.UUID {
	md5Str := fmt.Sprintf("%x",md5.Sum([]byte(str)))
	uuid := strfmt.UUID(md5Str[0:8] + "-" + md5Str[8:12] + "-" + md5Str[12:16] + "-" + md5Str[16:20] + "-" + md5Str[20:])
	return &uuid
}

func readSystemUUID() *strfmt.UUID {
	ret := strfmt.UUID(getDmiValue("system-uuid"))
	return &ret
}


func readMotherboadSerial() *strfmt.UUID {
	value := getDmiValue("baseboard-serial-number")
	if value == "" {
		log.Warn("Could not find motherboard serial number")
		return nil
	}
	return md5GenerateUUID(value)
}

func ReadId() *strfmt.UUID {

	ret := readMotherboadSerial()
	if ret == nil {
		ret = readSystemUUID()
	}
	return ret
}
