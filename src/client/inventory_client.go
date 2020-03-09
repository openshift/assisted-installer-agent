package client

import (
	"fmt"
	"github.com/filanov/bm-inventory/client"
	"github.com/ori-amizur/introspector/src/config"
	"net/http"
	"net/url"
)

func createUrl() string {
	return fmt.Sprintf("http://%s:%d/%s", config.GlobalConfig.TargetHost, config.GlobalConfig.TargetPort, client.DefaultBasePath)
}

type RequestRoundTripper struct {next http.RoundTripper}

func (rt *RequestRoundTripper)RoundTrip(req *http.Request) (*http.Response, error) {
	return rt.next.RoundTrip(req)
}


func CreateBmInventoryClient() *client.BMInventory {
	clientConfig := client.Config{}
	clientConfig.URL,_  = url.Parse(createUrl())
	clientConfig.Transport = &RequestRoundTripper{next: http.DefaultTransport}
	bmInventory := client.New(clientConfig)
	return bmInventory
}

