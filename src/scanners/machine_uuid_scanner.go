package scanners

import (
	bytes2 "bytes"
	"crypto/md5"
	"fmt"
	"github.com/go-openapi/strfmt"
	"github.com/ori-amizur/introspector/src/config"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strings"
)


func getDmiValue(keyword string) string {
	cmd := exec.Command("docker", "run", "--rm", "--privileged", config.GlobalConfig.DmidecodeImage, "dmidecode", "-s", keyword)
	var stdout, stderr bytes2.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Warnf("Error executing %s: %s", "docker", err.Error())
		return ""
	}
	return strings.TrimSpace(string(stdout.Bytes()))
}

func md5GenerateUUID(str string) *strfmt.UUID {
	md5Str := fmt.Sprintf("%x",md5.Sum([]byte(str)))
	uuid := strfmt.UUID(md5Str[0:8] + "-" + md5Str[8:12] + "-" + md5Str[12:16] + "-" + md5Str[16:20] + "-" + md5Str[20:])
	return &uuid
}

func readSystemUUID() *strfmt.UUID {
	ret := strfmt.UUID(strings.ToLower(getDmiValue("system-uuid")))
	return &ret
}


func readMotherboardSerial() *strfmt.UUID {
	value := getDmiValue("baseboard-serial-number")
	if value == "" {
		log.Warn("Could not find motherboard serial number")
		return nil
	}
	return md5GenerateUUID(value)
}

func ReadId() *strfmt.UUID {
	ret := readMotherboardSerial()
	if ret == nil {
		ret = readSystemUUID()
	}
	return ret
}
