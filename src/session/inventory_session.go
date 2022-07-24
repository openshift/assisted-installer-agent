package session

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"time"

	"github.com/PuerkitoBio/rehttp"
	"github.com/go-openapi/runtime"
	rtclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/pkg/errors"

	"github.com/openshift/assisted-service/client"
	"github.com/openshift/assisted-service/models"
	"github.com/openshift/assisted-service/pkg/auth"
	"github.com/openshift/assisted-service/pkg/requestid"
	"github.com/sirupsen/logrus"
)

var (
	minDelay = time.Duration(2) * time.Second
	maxDelay = time.Duration(10) * time.Second
	retries  = 3
)

func createUrl(inventoryUrl string) (*url.URL, error) {
	u, err := url.Parse(inventoryUrl)
	if err != nil {
		return nil, errors.Wrap(err, "Failed parsing inventory URL")
	}
	u.Path = path.Join(u.Path, client.DefaultBasePath)
	return u, nil
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

// HTMLConsumer handles cases where the response is not returned from the service
// itself, but from a middle agent such as an OCP router. The response is usually
// a standard HTML error page, but it is wrapped by the generated client as an
// applicative error object as defined by swagger.yaml.
// When the standard HTML consumer is handling such an error it yields a parsing
// error, therefore we replace the standard consumer with an application aware
// code.
func HTMLConsumer() runtime.Consumer {
	return runtime.ConsumerFunc(func(reader io.Reader, data interface{}) error {
		if reader == nil {
			return errors.New("HTMLConsumer requires a reader") // early exit
		}

		//read the response body
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(reader)
		if err != nil {
			return err
		}
		b := buf.Bytes()
		msg := string(b)

		//handle empty response body
		if len(b) == 0 {
			return nil
		}

		t := reflect.TypeOf(data)
		if data == nil || t.Kind() != reflect.Ptr {
			return fmt.Errorf("data should be a non nil pointer")
		}

		switch dt := data.(type) {
		case string:
			v := reflect.Indirect(reflect.ValueOf(data))
			v.SetString(msg)
		case encoding.TextUnmarshaler:
			return dt.UnmarshalText(b)
		case *models.Error:
			dt.Reason = swag.String(msg)
		case *models.InfraError:
			dt.Message = swag.String(msg)
		default:
			return fmt.Errorf("%+v (%T) is not supported by the Agent's Custom Consumer", data, data)
		}

		return nil
	})
}

func createBmInventoryClient(agentConfig *config.AgentConfig, inventoryUrl string, pullSecretToken string) (*client.AssistedInstall, error) {
	clientConfig := client.Config{}
	var err error
	clientConfig.URL, err = createUrl(inventoryUrl)
	if err != nil {
		return nil, err
	}

	var certs *x509.CertPool
	if agentConfig.InsecureConnection {
		logrus.Warn("Certificate verification is turned off. This is not recommended in production environments")
	} else {
		certs, err = readCACertificate(agentConfig)
		if err != nil {
			return nil, err
		}
	}

	transport := requestid.Transport(&http.Transport{
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
			InsecureSkipVerify: agentConfig.InsecureConnection,
			RootCAs:            certs,
		},
	})

	// Add retry settings
	tr := rehttp.NewTransport(
		transport,
		rehttp.RetryAll(
			rehttp.RetryMaxRetries(retries),
			rehttp.RetryTemporaryErr(),
		),
		rehttp.ExpJitterDelay(minDelay, maxDelay),
	)

	clientConfig.Transport = tr

	clientConfig.AuthInfo = auth.AgentAuthHeaderWriter(pullSecretToken)
	bmInventory := client.New(clientConfig)
	rtctransport := bmInventory.Transport.(*rtclient.Runtime)
	rtctransport.Consumers[runtime.HTMLMime] = HTMLConsumer()
	return bmInventory, nil
}

func readCACertificate(agentConfig *config.AgentConfig) (*x509.CertPool, error) {

	if agentConfig.CACertificatePath == "" {
		return nil, nil
	}

	caData, err := ioutil.ReadFile(agentConfig.CACertificatePath)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to load certificate: %s", agentConfig.CACertificatePath)
	}

	return pool, nil
}

func New(agentConfig *config.AgentConfig, inventoryUrl string, pullSecretToken string, log logrus.FieldLogger) (*InventorySession, error) {
	id := requestid.NewID()
	inventory, err := createBmInventoryClient(agentConfig, inventoryUrl, pullSecretToken)
	if err != nil {
		return nil, err
	}
	ret := InventorySession{
		ctx:    requestid.ToContext(context.Background(), id),
		logger: requestid.RequestIDLogger(log, id),
		client: inventory,
	}
	return &ret, nil
}
