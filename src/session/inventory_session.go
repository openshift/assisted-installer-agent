package session

import (
	"context"
	"fmt"
	"github.com/filanov/bm-inventory/client"
	"github.com/filanov/bm-inventory/pkg/requestid"
	"github.com/ori-amizur/introspector/src/config"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
)

func createUrl() string {
	return fmt.Sprintf("http://%s:%d/%s", config.GlobalConfig.TargetHost, config.GlobalConfig.TargetPort, client.DefaultBasePath)
}


type InventorySession struct {
	ctx context.Context
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

func createBmInventoryClient() *client.AssistedInstall {
	clientConfig := client.Config{}
	clientConfig.URL,_  = url.Parse(createUrl())
	clientConfig.Transport = requestid.Transport(http.DefaultTransport)
	bmInventory := client.New(clientConfig)
	return bmInventory
}

func New() *InventorySession {
	id := requestid.NewID()
	ret := InventorySession{
		ctx:    requestid.ToContext(context.Background(), id),
		logger: requestid.RequestIDLogger(logrus.StandardLogger(), id),
		client: createBmInventoryClient(),
	}
	return &ret
}
