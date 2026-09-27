package connectivity_check

import (
	"encoding/json"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

type connectivity struct {
	dryRunConfig *config.DryRunConfig
}

func (c *connectivity) checkers() []Checker {
	if c.dryRunConfig.DryRunEnabled {
		return []Checker{
			&dryL2Checker{},
			&dryL3Checker{},
		}
	} else {
		e := newExecuter()
		return []Checker{
			&pingChecker{executer: e},
			&arpingChecker{executer: e},
			&nmapChecker{executer: e},
			&mtuChecker{executer: e},
		}
	}
}

func (c *connectivity) connectivityCheck(args ...string) (stdout string, stderr string, exitCode int) {
	if len(args) != 1 {
		return "", "Expecting exactly 1 argument for connectivity command", -1
	}
	params := models.ConnectivityCheckParams{}
	err := json.Unmarshal([]byte(args[0]), &params)
	if err != nil {
		log.Warnf("Error unmarshalling json %s: %s", args[0], err.Error())
		return "", err.Error(), -1
	}
	nics := getOutgoingNics(c.dryRunConfig, nil)

	d := &connectivityRunner{checkers: c.checkers()}
	ret, err := d.Run(params, nics)
	if err != nil {
		log.WithError(err).Warn("Could not run connectivity check")
		return "", err.Error(), -1
	}
	bytes, err := json.Marshal(&ret)
	if err != nil {
		log.Warnf("Could not marshal json: %s", err.Error())
		return "", err.Error(), -1
	}
	return string(bytes), "", 0
}

func ConnectivityCheck(dryRunConfig *config.DryRunConfig, args ...string) (string, string, int) {
	c := &connectivity{
		dryRunConfig: dryRunConfig,
	}
	return c.connectivityCheck(args...)
}
