package utils

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func SetDebugLogLevel() {
	log.SetLevel(logrus.DebugLevel)
}

func FletcherChecksum8(inp string) uint8 {
	var ckA, ckB uint8
	for i := 0; i < len(inp); i++ {
		ckA = (ckA + inp[i]) % 0xf
		ckB = (ckB + ckA) % 0xf
	}
	return (ckB << 4) | ckA
}

func ShortHostname() (shortName string, err error) {
	var hostname string

	if filePath, ok := os.LookupEnv("RUNTIMECFG_HOSTNAME_PATH"); ok {
		dat, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.WithFields(logrus.Fields{
				"filePath": filePath,
			}).Error("Failed to read file")
			return "", err
		}
		hostname = strings.TrimSuffix(string(dat), "\n")
		log.WithFields(logrus.Fields{
			"hostname": hostname,
			"file":     filePath,
		}).Debug("Hostname retrieved from a file")
	} else {
		hostname, err = os.Hostname()
		if err != nil {
			panic(err)
		}
		log.WithFields(logrus.Fields{
			"hostname": hostname,
		}).Debug("Hostname retrieved from OS")
	}
	splitHostname := strings.SplitN(hostname, ".", 2)
	shortName = splitHostname[0]
	return shortName, err
}

func IsKubernetesHealthy(port uint16) (bool, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}
	resp, err := client.Get(fmt.Sprintf("https://localhost:%d/readyz", port))
	if err != nil {
		return false, err
	}
	defer client.CloseIdleConnections()
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	return string(body) == "ok", nil
}

func AlarmStabilization(cur_alrm bool, cur_defect bool, consecutive_ctr uint8, on_threshold uint8, off_threshold uint8) (bool, uint8) {
	var new_alrm bool = cur_alrm
	var threshold uint8

	if cur_alrm != cur_defect {
		consecutive_ctr++
		if cur_alrm {
			threshold = off_threshold
		} else {
			threshold = on_threshold
		}
		if consecutive_ctr >= threshold {
			new_alrm = !cur_alrm
			consecutive_ctr = 0
		}
	} else {
		consecutive_ctr = 0
	}
	return new_alrm, consecutive_ctr
}

func GetFileMd5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil
}
