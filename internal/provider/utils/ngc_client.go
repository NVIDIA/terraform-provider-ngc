package utils

import (
	"net/http"
	"sync"
)

type NGCClient struct {
	NgcEndpoint string
	NgcApiKey   string
	NgcOrg      string
	NgcTeam     string
	HttpClient  *http.Client
}

var nvcfClient *NVCFClient = nil
var nvcfClientOnce sync.Once

func (c *NGCClient) NVCFClient() *NVCFClient {
	nvcfClientOnce.Do(func() {
		nvcfClient = &NVCFClient{c.NgcEndpoint, c.NgcApiKey, c.NgcOrg, c.NgcTeam, c.HttpClient}
	})
	return nvcfClient
}
