package commands

import (
	"encoding/json"
	"github.com/filanov/bm-inventory/models"
	log "github.com/sirupsen/logrus"
)

type IPAddressConnectivity {

}

func ConnectivityCheck(input string) (string, error) {
	params := make(models.ConnectivityCheckParams, 0)
	err := json.Unmarshal([]byte(input), &params)
	if err != nil {
		log.Warnf("Error unmarshalling json %s: %s", input, err.Error())
		return "", err
	}
	for _, node := range params {

	}
}
