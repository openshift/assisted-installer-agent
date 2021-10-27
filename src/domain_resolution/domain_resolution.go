package domain_resolution

import (
	"encoding/json"
	"net"

	"github.com/go-openapi/strfmt"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

//go:generate mockery -name DomainResolutionDependencies -inpkg
type DomainResolutionDependencies interface {
	Resolve(domain string) (ips []net.IP, err error)
}

type DomainResolver struct{}

func (e *DomainResolver) Resolve(domain string) (ips []net.IP, err error) {
	ips, err = net.LookupIP(domain)

	// No need to return error in case domain was not found
	// It is expected answer and service will handle it
	if err != nil {
		if e, ok := err.(*net.DNSError); ok && e.IsNotFound  {
			err = nil
		}
	}
	return ips, err
}

func handleDomainResolution(resolver DomainResolutionDependencies, log logrus.FieldLogger, domain string) models.DomainResolutionResponseDomain {
	result := models.DomainResolutionResponseDomain{
		DomainName:    &domain,
		IPV4Addresses: make([]strfmt.IPv4, 0),
		IPV6Addresses: make([]strfmt.IPv6, 0),
	}

	ips, err := resolver.Resolve(domain)
	if err != nil {
		log.WithError(err).Errorf("error occurred during domain resolution of %s", domain)
		return result
	}

	for _, ip := range ips {
		if ip.To4() != nil {
			result.IPV4Addresses = append(result.IPV4Addresses, strfmt.IPv4(ip.String()))
		} else if ip.To16() != nil {
			result.IPV6Addresses = append(result.IPV6Addresses, strfmt.IPv6(ip.String()))
		} else {
			log.Errorf("IP address %v of %s is neither IPv4 nor IPv6, ignoring", ip, domain)
		}
	}

	return result
}

func Run(requestStr string, resolver DomainResolutionDependencies, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var request models.DomainResolutionRequest

	err := json.Unmarshal([]byte(requestStr), &request)
	if err != nil {
		log.WithError(err).Errorf("Failed to unmarshal domain resolution request string %s", requestStr)
		return "", err.Error(), -1
	}

	response := models.DomainResolutionResponse{
		Resolutions: make([]*models.DomainResolutionResponseDomain, 0),
	}

	for _, domain := range request.Domains {
		if domain.DomainName == nil {
			return "", "Every domain in a domain request must have a domain name field", -1
		}

		resolution := handleDomainResolution(resolver, log, *domain.DomainName)
		response.Resolutions = append(response.Resolutions, &resolution)
	}

	b, err := json.Marshal(&response)
	if err != nil {
		log.WithError(err).Errorf("Failed to domain resolution availability response %v", response)
		return "", err.Error(), -1
	}
	return string(b), "", 0
}
