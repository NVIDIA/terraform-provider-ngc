//  SPDX-FileCopyrightText: Copyright (c) 2024 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
//  SPDX-License-Identifier: LicenseRef-NvidiaProprietary

//  NVIDIA CORPORATION, its affiliates and licensors retain all intellectual
//  property and proprietary rights in and to this material, related
//  documentation and any modifications thereto. Any use, reproduction,
//  disclosure or distribution of this material and related documentation
//  without an express license agreement from NVIDIA CORPORATION or
//  its affiliates is strictly prohibited.

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

// Terraform allows several aliased instances of the same provider, each with
// its own credentials. NVCFClient() must therefore reflect the NGCClient it was
// called on. A package-level singleton would hand the second alias the first
// alias's API key, org and endpoint — sending one tenant's key to another
// tenant's endpoint.
func TestNGCClient_NVCFClient_PerInstanceCredentials(t *testing.T) {
	first := &NGCClient{
		NgcEndpoint: "https://first.example.invalid",
		NgcApiKey:   "FIRST_API_KEY",
		NgcOrg:      "FIRST_ORG",
		NgcTeam:     "FIRST_TEAM",
		HttpClient:  http.DefaultClient,
	}
	second := &NGCClient{
		NgcEndpoint: "https://second.example.invalid",
		NgcApiKey:   "SECOND_API_KEY",
		NgcOrg:      "SECOND_ORG",
		NgcTeam:     "SECOND_TEAM",
		HttpClient:  http.DefaultClient,
	}

	// Order matters: the first call is what a singleton would freeze in place.
	firstClient := first.NVCFClient()
	secondClient := second.NVCFClient()

	if firstClient.NgcApiKey != "FIRST_API_KEY" {
		t.Errorf("first alias got API key %q, want FIRST_API_KEY", firstClient.NgcApiKey)
	}
	if secondClient.NgcApiKey != "SECOND_API_KEY" {
		t.Errorf("second alias got API key %q, want SECOND_API_KEY (credential cross-contamination)", secondClient.NgcApiKey)
	}
	if secondClient.NgcEndpoint != "https://second.example.invalid" {
		t.Errorf("second alias got endpoint %q, want the second alias's endpoint", secondClient.NgcEndpoint)
	}
	if secondClient.NgcOrg != "SECOND_ORG" {
		t.Errorf("second alias got org %q, want SECOND_ORG", secondClient.NgcOrg)
	}
	if firstClient == secondClient {
		t.Error("both aliases share one *NVCFClient; each provider instance must get its own")
	}
}
