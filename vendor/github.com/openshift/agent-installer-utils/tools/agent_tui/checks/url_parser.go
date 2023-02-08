package checks

import (
	"fmt"
	"net/url"
	"strings"
)

// URL may be missing the scheme://
func ParseHostnameFromURL(urlString string) (string, error) {
	urlWithScheme := urlString
	if !strings.Contains(urlString, "://") {
		// missing scheme, add one to allow url.Parse to work correctly
		urlWithScheme = "https://" + urlWithScheme
	}
	parsedUrl, err := url.Parse(urlWithScheme)
	if err != nil {
		return "", err
	}
	return parsedUrl.Hostname(), nil
}

// Returns the url without path
func ParseSchemeHostnamePortFromURL(urlString string, schemeIfMissing string) (string, error) {
	urlWithScheme := urlString
	if !strings.Contains(urlString, "://") {
		// missing scheme, add one to allow url.Parse to work correctly
		urlWithScheme = schemeIfMissing + urlWithScheme
	}
	parsedUrl, err := url.Parse(urlWithScheme)
	if err != nil {
		return "", err
	}

	if parsedUrl.Port() != "" {
		return fmt.Sprintf("%s://%s:%s", parsedUrl.Scheme, parsedUrl.Hostname(), parsedUrl.Port()), nil
	} else {
		return fmt.Sprintf("%s://%s", parsedUrl.Scheme, parsedUrl.Hostname()), nil
	}
}
