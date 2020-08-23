package session

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/openshift/assisted-installer-agent/src/config"

	"github.com/openshift/assisted-service/client"
	"github.com/openshift/assisted-service/pkg/auth"
	"github.com/openshift/assisted-service/pkg/requestid"
	"github.com/sirupsen/logrus"
)

func createUrl(inventoryUrl string) string {
	return fmt.Sprintf("%s/%s", inventoryUrl, client.DefaultBasePath)
}

type InventorySession struct {
	ctx    context.Context
	logger logrus.FieldLogger
	client *client.AssistedInstall
}

func (i *InventorySession) Context() context.Context {
	return i.ctx
}

func (i *InventorySession) Logger() logrus.FieldLogger {
	return i.logger
}

func (i *InventorySession) Client() *client.AssistedInstall {
	return i.client
}

func createBmInventoryClient(inventoryUrl string, pullSecretToken string) (*client.AssistedInstall, error) {
	clientConfig := client.Config{}
	var err error
	clientConfig.URL, err = url.ParseRequestURI(createUrl(inventoryUrl))
	if err != nil {
		return nil, err
	}

	var certs *x509.CertPool
	if config.GlobalAgentConfig.InsecureConnection {
		logrus.Warn("Certificate verification is turned off. This is not recommended in production environments")
	} else {
		certs, err = readCACertificate()
		if err != nil {
			return nil, err
		}
	}

	clientConfig.Transport = requestid.Transport(&http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.GlobalAgentConfig.InsecureConnection,
			RootCAs:            certs,
		},
	})

	clientConfig.AuthInfo = auth.AgentAuthHeaderWriter(pullSecretToken)
	bmInventory := client.New(clientConfig)
	return bmInventory, nil
}

func readCACertificate() (*x509.CertPool, error) {

	if config.GlobalAgentConfig.CACertificatePath == "" {
		return nil, nil
	}

	caData, err := ioutil.ReadFile(config.GlobalAgentConfig.CACertificatePath)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to load certificate: %s", config.GlobalAgentConfig.CACertificatePath)
	}

	return pool, nil
}

func New(inventoryUrl string, pullSecretToken string) (*InventorySession, error) {
	id := requestid.NewID()
	inventory, err := createBmInventoryClient(inventoryUrl, pullSecretToken)
	if err != nil {
		return nil, err
	}
	ret := InventorySession{
		ctx:    requestid.ToContext(context.Background(), id),
		logger: requestid.RequestIDLogger(logrus.StandardLogger(), id),
		client: inventory,
	}
	return &ret, nil
}
