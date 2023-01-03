package vips_verifier

import (
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-installer-agent/src/util/nmap"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate mockery --name Executer --inpackage
type Executer interface {
	Execute(command string, args ...string) (stdout string, stderr string, exitCode int)
}

type VipVerifier interface {
	verifyVip(vipToVerify *models.VerifyVip) (*models.VerifiedVip, error)
}

type executer struct{}

func (e *executer) Execute(command string, args ...string) (stdout string, stderr string, exitCode int) {
	return util.Execute(command, args...)
}

func bailOut(errStr string) (stdout string, stderr string, exitCode int) {
	logrus.Errorf("vip-verifier: %s", errStr)
	return "", errStr, -1
}

type dryVipVerifier struct{}

func (*dryVipVerifier) verifyVip(vipToVerify *models.VerifyVip) (*models.VerifiedVip, error) {
	return &models.VerifiedVip{
		Verification: models.NewVipVerification(models.VipVerificationSucceeded),
		Vip:          vipToVerify.Vip,
		VipType:      vipToVerify.VipType,
	}, nil
}

type vipVerifier struct {
	exe Executer
}

func (v *vipVerifier) verifyVip(vipToVerify *models.VerifyVip) (*models.VerifiedVip, error) {
	var (
		o, e     string
		exitCode int
	)
	vip := string(vipToVerify.Vip)
	if util.IsIPv4Addr(vip) {
		o, e, exitCode = v.exe.Execute("nmap", "-sn", "-n", "-oX", "-", vip)
	} else {
		o, e, exitCode = v.exe.Execute("nmap", "-6", "-sn", "-n", "-oX", "-", vip)
	}
	if exitCode != 0 {
		return nil, errors.Errorf("nmap exited with code %d: %s", exitCode, e)
	}
	var nmaprun nmap.Nmaprun
	if err := xml.Unmarshal([]byte(o), &nmaprun); err != nil {
		logrus.WithError(err).Warn("XML Unmarshal")
		return nil, errors.Wrap(err, "XML Unmarshal")
	}
	ret := &models.VerifiedVip{
		Vip:          vipToVerify.Vip,
		VipType:      vipToVerify.VipType,
		Verification: models.NewVipVerification(models.VipVerificationSucceeded),
	}
	for _, h := range nmaprun.Hosts {
		if h.Status.State == "up" {
			for _, a := range h.Addresses {
				if a.AddrType == "ipv4" || a.AddrType == "ipv6" {
					ret.Verification = models.NewVipVerification(models.VipVerificationFailed)
					break
				}
			}
		}
	}
	return ret, nil
}

func verifyVips(verifier VipVerifier, arg string) (stdout string, stderr string, exitCode int) {
	var verifyVipsRequest models.VerifyVipsRequest
	if err := json.Unmarshal([]byte(arg), &verifyVipsRequest); err != nil {
		return bailOut(fmt.Sprintf("failed to unmarshal argument: %s", err.Error()))
	}
	var verifyVipsResponse models.VerifyVipsResponse
	for _, vipToVerify := range verifyVipsRequest {
		verifiedVip, err := verifier.verifyVip(vipToVerify)
		if err != nil {
			return bailOut(fmt.Sprintf("failed to verify vip %s: %s", vipToVerify.Vip, err.Error()))
		}
		verifyVipsResponse = append(verifyVipsResponse, verifiedVip)
	}
	b, err := json.Marshal(&verifyVipsResponse)
	if err != nil {
		return bailOut(fmt.Sprintf("Marshal response: %s", err.Error()))
	}
	return string(b), "", 0
}

func VerifyVips(dryRunConfig *config.DryRunConfig, _ string, args ...string) (stdout string, stderr string, exitCode int) {
	if len(args) != 1 {
		return bailOut(fmt.Sprintf("expected 1 argument.  Received %d", len(args)))
	}
	var verifier VipVerifier = &vipVerifier{exe: &executer{}}
	if dryRunConfig.DryRunEnabled {
		verifier = &dryVipVerifier{}
	}
	return verifyVips(verifier, args[0])
}
