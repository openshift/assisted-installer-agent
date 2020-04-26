package client

import (
	"context"
	"fmt"
	"github.com/filanov/bm-inventory/client"
	"github.com/filanov/bm-inventory/pkg/requestid"
	"github.com/ori-amizur/introspector/src/config"
	"github.com/ori-amizur/introspector/src/util"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
)

func createUrl() string {
	return fmt.Sprintf("http://%s:%d/%s", config.GlobalConfig.TargetHost, config.GlobalConfig.TargetPort, client.DefaultBasePath)
}

func CreateBmInventoryClient() *client.BMInventory {
	clientConfig := client.Config{}
	clientConfig.URL,_  = url.Parse(createUrl())
	clientConfig.Transport = requestid.Transport(http.DefaultTransport)
	bmInventory := client.New(clientConfig)
	return bmInventory
}

func NewContext() context.Context {
	id := requestid.NewID()
	ctx := util.WithLogger(context.Background(), requestid.RequestIDLogger(logrus.StandardLogger(), id))
	return requestid.ToContext(ctx, id)
}
