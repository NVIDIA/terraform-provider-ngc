//go:build unittest
// +build unittest

package utils

import (
	"net/http"
	"reflect"
	"testing"
)

func TestNGCClient_NVCFClient(t *testing.T) {
	t.Parallel()

	testHttpClient := http.DefaultClient

	type fields struct {
		NgcEndpoint string
		NgcApiKey   string
		NgcOrg      string
		NgcTeam     string
		HttpClient  *http.Client
	}
	tests := []struct {
		name   string
		fields fields
		want   *NVCFClient
	}{
		{
			name: `NVCFClientInitSucceed`,
			fields: fields{
				NgcEndpoint: "MOCK_ENDPOINT",
				NgcApiKey:   "MOCK_API",
				NgcOrg:      "MOCK_ORG",
				NgcTeam:     "MOCK_TEAM",
				HttpClient:  testHttpClient,
			},
			want: &NVCFClient{
				NgcEndpoint: "MOCK_ENDPOINT",
				NgcApiKey:   "MOCK_API",
				NgcOrg:      "MOCK_ORG",
				NgcTeam:     "MOCK_TEAM",
				HttpClient:  testHttpClient,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &NGCClient{
				NgcEndpoint: tt.fields.NgcEndpoint,
				NgcApiKey:   tt.fields.NgcApiKey,
				NgcOrg:      tt.fields.NgcOrg,
				NgcTeam:     tt.fields.NgcTeam,
				HttpClient:  tt.fields.HttpClient,
			}
			if got := c.NVCFClient(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NGCClient.NVCFClient() = %v, want %v", got, tt.want)
			}
		})
	}
}
