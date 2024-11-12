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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var mockOrg = "MOCK_ORG"
var mockTeam = "MOCK_TEAM"
var mockApiKey = "MOCK_API_KEY"
var mockEndpoint = "https://MOCK_API_ENDPOINT"
var nvcfRequestHeaders = map[string]string{
	"Authorization": "Bearer MOCK_API_KEY",
	"Content-Type":  "application/json",
}
var mockFunctionID = "033c9664-f5b0-4bd2-8918-5aab085fc8db"
var mockVersionID = "f0cc4c95-108c-471a-b52c-a2bd5c0024c2"
var mockHelmBasedFunctionInfo = fmt.Sprintf(
	`{
		"id": "%s",
		"ncaId": "SfDTycz_Y81Iq7rCtGXj4gy93huIjvzQ3ZtNvumZywg",
		"versionId": "%s",
		"name": "mock-helm-function",
		"status": "INACTIVE",
		"inferenceUrl": "/",
		"inferencePort": 50051,
		"helmChart": "mock",
		"helmChartServiceName": "entry",
		"healthUri": "mock",
		"apiBodyFormat": "CUSTOM",
		"healthUri": "/",
		"createdAt": "2024-03-13T09:04:20.377756757Z",
		"activeInstances": []
	}`,
	mockFunctionID,
	mockVersionID,
)
var mockContainerBasedFunctionInfo = fmt.Sprintf(`
	{
		"id": "%s",
		"ncaId": "SfDTycz_Y81Iq7rCtGXj4gy93huIjvzQ3ZtNvumZywg",
		"versionId": "%s",
		"name": "mock-container-function",
		"status": "INACTIVE",
		"inferenceUrl": "/",
		"inferencePort": 50051,
		"containerImage": "nvcr.io/lzzr0aktntgj/coreapi-service:latest-dev",
		"apiBodyFormat": "CUSTOM",
		"healthUri": "/",
		"createdAt": "2024-03-13T09:04:20.377756757Z",
		"activeInstances": []
	}
	`,
	mockFunctionID,
	mockVersionID,
)

var mockDeploymentSpecification = fmt.Sprintf(`
	{
		"gpu": "L40",
		"backend": "GFN",
		"maxInstances": 1,
		"minInstances": 1,
		"instanceType": "gl40_1.br20_2xlarge",
		"maxRequestConcurrency": 1,
		"configuration": "{\"image\":{\"repository\":\"nvcr.io/shhh2i6mga69/devinfra/fastapi_echo_sample\",\"tag\":\"latest\"}}"
	}`)
var mockFunctionDeploymentInfo = fmt.Sprintf(
	`
	{
		"deployment" : {
			"functionId": "%s",
			"functionVersionID": "%s",
			"ncaId": "SfDTycz_Y81Iq7rCtGXj4gy93huIjvzQ3ZtNvumZywg",
			"functionStatus": "DEPLOYING",
			"requestQueueUrl": "https://sqs.us-west-2.amazonaws.com/052277528122/gdn-strap-dynamic_SfDTycz-Y81Iq7rCt_6cf20357-b6c9-459e-ae36-34b22319b7e4.fifo",
			"deploymentSpecifications": [%s]
		}
	}
	`,
	mockFunctionID,
	mockVersionID,
	mockDeploymentSpecification,
)

var mockFunctionDeploymentFailedInfo = fmt.Sprintf(
	`
	{
		"deployment" : {
			"functionId": "%s",
			"functionVersionID": "%s",
			"ncaId": "SfDTycz_Y81Iq7rCtGXj4gy93huIjvzQ3ZtNvumZywg",
			"functionStatus": "FAILED",
			"requestQueueUrl": "https://sqs.us-west-2.amazonaws.com/052277528122/gdn-strap-dynamic_SfDTycz-Y81Iq7rCt_6cf20357-b6c9-459e-ae36-34b22319b7e4.fifo",
			"deploymentSpecifications": [%s]
		}
	}
	`,
	mockFunctionID,
	mockVersionID,
	mockDeploymentSpecification,
)

var mockFunctionDeploymentActiveInfo = fmt.Sprintf(
	`
	{
		"deployment" : {
			"functionId": "%s",
			"functionVersionID": "%s",
			"ncaId": "SfDTycz_Y81Iq7rCtGXj4gy93huIjvzQ3ZtNvumZywg",
			"functionStatus": "ACTIVE",
			"requestQueueUrl": "https://sqs.us-west-2.amazonaws.com/052277528122/gdn-strap-dynamic_SfDTycz-Y81Iq7rCt_6cf20357-b6c9-459e-ae36-34b22319b7e4.fifo",
			"deploymentSpecifications": [%s]
		}
	}
	`,
	mockFunctionID,
	mockVersionID,
	mockDeploymentSpecification,
)

var mockErrorDetail = "Validation failed - [All allocated GPU instances in use - contact your support team]"
var mockErrorResponse = fmt.Sprintf(
	`
	{
		"requestStatus": {
			"statusCode": "INVALID_REQUEST",
			"statusDescription": "%s",
			"requestId": "a3023cc6-2705972"
		}
	}
	`,
	mockErrorDetail,
)

type mockRoundTripper struct {
	t        *testing.T
	request  *http.Request
	response *http.Response
}

func (rt *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	assert.Equal(rt.t, req.Body, rt.request.Body)
	assert.Equal(rt.t, req.Method, rt.request.Method)
	assert.Equal(rt.t, req.URL.Path, rt.request.URL.Path)
	assert.Equal(rt.t, req.Header, rt.request.Header)
	return rt.response, nil
}

func GenerateHttpClientMockRoundTripper(t *testing.T, target string, method string, reqHeaders map[string]string, req any, resp string, respCode int) *mockRoundTripper {
	var expectedRequest *http.Request
	if req != nil {
		payloadBuf := new(bytes.Buffer)
		json.NewEncoder(payloadBuf).Encode(req)
		expectedRequest = httptest.NewRequest(method, target, payloadBuf)
	} else {
		expectedRequest = httptest.NewRequest(method, target, http.NoBody)
	}

	for k, v := range reqHeaders {
		expectedRequest.Header.Set(k, v)
	}

	recorder := httptest.NewRecorder()
	recorder.Header().Add("Content-Type", "application/json")
	recorder.WriteString(resp)
	expectedResponse := recorder.Result()
	expectedResponse.StatusCode = respCode
	return &mockRoundTripper{t, expectedRequest, expectedResponse}
}

func TestNVCFClient_NvcfEndpoint(t *testing.T) {
	t.Parallel()

	httpClient := http.DefaultClient

	type fields struct {
		NgcEndpoint string
		NgcApiKey   string
		NgcOrg      string
		NgcTeam     string
		HttpClient  *http.Client
	}
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "GetEndpointWithTeam",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient:  httpClient,
			},
			want: fmt.Sprintf("%s/v2/orgs/%s/teams/%s", mockEndpoint, mockOrg, mockTeam),
		},
		{
			name: "GetEndpoint",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				HttpClient:  httpClient,
			},
			want: fmt.Sprintf("%s/v2/orgs/%s", mockEndpoint, mockOrg),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &NVCFClient{
				NgcEndpoint: tt.fields.NgcEndpoint,
				NgcApiKey:   tt.fields.NgcApiKey,
				NgcOrg:      tt.fields.NgcOrg,
				NgcTeam:     tt.fields.NgcTeam,
				HttpClient:  tt.fields.HttpClient,
			}
			if got := c.NvcfEndpoint(tt.args.in0); got != tt.want {
				t.Errorf("NVCFClient.NvcfEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNVCFClient_CreateNvidiaCloudFunction(t *testing.T) {
	t.Parallel()

	createContainerBasedNvidiaCloudFunctionMockRespRaw := mockContainerBasedFunctionInfo
	var createContainerBasedNvidiaCloudFunctionMockResp CreateNvidiaCloudFunctionResponse
	json.Unmarshal([]byte(createContainerBasedNvidiaCloudFunctionMockRespRaw), &createContainerBasedNvidiaCloudFunctionMockResp)

	createContainerBasedNvidiaCloudFunctionReq := CreateNvidiaCloudFunctionRequest{
		ContainerImage: "nvcr.io/lzzr0aktntgj/coreapi-service:latest-dev",
		InferencePort:  50051,
		InferenceUrl:   "/",
		HealthUri:      "/",
		APIBodyFormat:  "CUSTOM",
		FunctionName:   "mock-container-function",
	}

	createHelmBasedNvidiaCloudFunctionMockRespRaw := mockHelmBasedFunctionInfo
	var createHelmBasedNvidiaCloudFunctionMockResp CreateNvidiaCloudFunctionResponse
	json.Unmarshal([]byte(createHelmBasedNvidiaCloudFunctionMockRespRaw), &createHelmBasedNvidiaCloudFunctionMockResp)

	createHelmBasedNvidiaCloudFunctionReq := CreateNvidiaCloudFunctionRequest{
		HelmChart:            "mock",
		InferencePort:        50051,
		HelmChartServiceName: "entry",
		InferenceUrl:         "/",
		HealthUri:            "/",
		APIBodyFormat:        "CUSTOM",
		FunctionName:         "mock-helm-function",
	}
	type fields struct {
		NgcEndpoint string
		NgcApiKey   string
		NgcOrg      string
		NgcTeam     string
		HttpClient  *http.Client
	}
	type args struct {
		ctx        context.Context
		functionID string
		req        CreateNvidiaCloudFunctionRequest
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantResp   *CreateNvidiaCloudFunctionResponse
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "CreateContainerBasedNvidiaCloudFunction",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/functions", mockEndpoint, mockOrg, mockTeam),
						http.MethodPost,
						nvcfRequestHeaders,
						createContainerBasedNvidiaCloudFunctionReq,
						createContainerBasedNvidiaCloudFunctionMockRespRaw,
						200,
					),
				},
			},
			args: args{
				ctx:        context.Background(),
				functionID: "",
				req: CreateNvidiaCloudFunctionRequest{
					ContainerImage: "nvcr.io/lzzr0aktntgj/coreapi-service:latest-dev",
					InferencePort:  50051,
					InferenceUrl:   "/",
					HealthUri:      "/",
					APIBodyFormat:  "CUSTOM",
					FunctionName:   "mock-container-function",
				},
			},
			wantResp:   &createContainerBasedNvidiaCloudFunctionMockResp,
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "CreateHelmBasedNvidiaCloudFunction",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/functions", mockEndpoint, mockOrg, mockTeam),
						http.MethodPost,
						nvcfRequestHeaders,
						createHelmBasedNvidiaCloudFunctionReq,
						createHelmBasedNvidiaCloudFunctionMockRespRaw,
						200,
					),
				},
			},
			args: args{
				ctx:        context.Background(),
				functionID: "",
				req: CreateNvidiaCloudFunctionRequest{
					HelmChart:            "mock",
					InferencePort:        50051,
					HelmChartServiceName: "entry",
					InferenceUrl:         "/",
					HealthUri:            "/",
					APIBodyFormat:        "CUSTOM",
					FunctionName:         "mock-helm-function",
				},
			},
			wantResp:   &createContainerBasedNvidiaCloudFunctionMockResp,
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "CreateContainerBasedNvidiaCloudFunctionVersion",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/functions/%s/versions", mockEndpoint, mockOrg, mockTeam, mockFunctionID),
						http.MethodPost,
						nvcfRequestHeaders,
						createContainerBasedNvidiaCloudFunctionReq,
						createContainerBasedNvidiaCloudFunctionMockRespRaw,
						200,
					),
				},
			},
			args: args{
				ctx:        context.Background(),
				functionID: mockFunctionID,
				req: CreateNvidiaCloudFunctionRequest{
					ContainerImage: "nvcr.io/lzzr0aktntgj/coreapi-service:latest-dev",
					InferencePort:  50051,
					InferenceUrl:   "/",
					HealthUri:      "/",
					APIBodyFormat:  "CUSTOM",
					FunctionName:   "mock-container-function",
				},
			},
			wantResp:   &createContainerBasedNvidiaCloudFunctionMockResp,
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "CreateContainerBasedNvidiaCloudFunctionVersionFailed",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/functions/%s/versions", mockEndpoint, mockOrg, mockTeam, mockFunctionID),
						http.MethodPost,
						nvcfRequestHeaders,
						createContainerBasedNvidiaCloudFunctionReq,
						mockErrorResponse,
						400,
					),
				},
			},
			args: args{
				ctx:        context.Background(),
				functionID: mockFunctionID,
				req: CreateNvidiaCloudFunctionRequest{
					ContainerImage: "nvcr.io/lzzr0aktntgj/coreapi-service:latest-dev",
					InferencePort:  50051,
					InferenceUrl:   "/",
					HealthUri:      "/",
					APIBodyFormat:  "CUSTOM",
					FunctionName:   "mock-container-function",
				},
			},
			wantResp:   &CreateNvidiaCloudFunctionResponse{},
			wantErr:    true,
			wantErrMsg: mockErrorDetail,
		},
		{
			name: "CreateHelmBasedNvidiaCloudFunctionVersion",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/functions/%s/versions", mockEndpoint, mockOrg, mockTeam, mockFunctionID),
						http.MethodPost,
						nvcfRequestHeaders,
						createHelmBasedNvidiaCloudFunctionReq,
						createHelmBasedNvidiaCloudFunctionMockRespRaw,
						200,
					),
				},
			},
			args: args{
				ctx:        context.Background(),
				functionID: mockFunctionID,
				req: CreateNvidiaCloudFunctionRequest{
					HelmChart:            "mock",
					InferencePort:        50051,
					HelmChartServiceName: "entry",
					InferenceUrl:         "/",
					HealthUri:            "/",
					APIBodyFormat:        "CUSTOM",
					FunctionName:         "mock-helm-function",
				},
			},
			wantResp: &createContainerBasedNvidiaCloudFunctionMockResp,
			wantErr:  false,
		},
		{
			name: "CreateHelmBasedNvidiaCloudFunctionVersionFailed",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/functions/%s/versions", mockEndpoint, mockOrg, mockTeam, mockFunctionID),
						http.MethodPost,
						nvcfRequestHeaders,
						createHelmBasedNvidiaCloudFunctionReq,
						mockErrorResponse,
						500,
					),
				},
			},
			args: args{
				ctx:        context.Background(),
				functionID: mockFunctionID,
				req: CreateNvidiaCloudFunctionRequest{
					HelmChart:            "mock",
					InferencePort:        50051,
					HelmChartServiceName: "entry",
					InferenceUrl:         "/",
					HealthUri:            "/",
					APIBodyFormat:        "CUSTOM",
					FunctionName:         "mock-helm-function",
				},
			},
			wantResp:   &CreateNvidiaCloudFunctionResponse{},
			wantErr:    true,
			wantErrMsg: mockErrorDetail,
		},
		{
			name: "CreateHelmBasedNvidiaCloudFunctionVersionUnauthorized",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/functions/%s/versions", mockEndpoint, mockOrg, mockTeam, mockFunctionID),
						http.MethodPost,
						nvcfRequestHeaders,
						createHelmBasedNvidiaCloudFunctionReq,
						"",
						401,
					),
				},
			},
			args: args{
				ctx:        context.Background(),
				functionID: mockFunctionID,
				req: CreateNvidiaCloudFunctionRequest{
					HelmChart:            "mock",
					InferencePort:        50051,
					HelmChartServiceName: "entry",
					InferenceUrl:         "/",
					HealthUri:            "/",
					APIBodyFormat:        "CUSTOM",
					FunctionName:         "mock-helm-function",
				},
			},
			wantResp:   &CreateNvidiaCloudFunctionResponse{},
			wantErr:    true,
			wantErrMsg: "not authenticated",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &NVCFClient{
				NgcEndpoint: tt.fields.NgcEndpoint,
				NgcApiKey:   tt.fields.NgcApiKey,
				NgcOrg:      tt.fields.NgcOrg,
				NgcTeam:     tt.fields.NgcTeam,
				HttpClient:  tt.fields.HttpClient,
			}
			gotResp, err := c.CreateNvidiaCloudFunction(tt.args.ctx, tt.args.functionID, tt.args.req)
			if (err != nil) != tt.wantErr || ((err != nil) && err.Error() != tt.wantErrMsg) {
				t.Errorf("NVCFClient.CreateNvidiaCloudFunction() error = %v, wantErr %v, wantErrMsg %v", err, tt.wantErr, tt.wantErrMsg)
				return
			}
			if !reflect.DeepEqual(gotResp, tt.wantResp) {
				t.Errorf("NVCFClient.CreateNvidiaCloudFunction() = %v, want %v", gotResp, tt.wantResp)
			}
		})
	}
}

func TestNVCFClient_ListNvidiaCloudFunctionVersions(t *testing.T) {
	t.Parallel()

	listNvidiaCloudFunctionVersionsMockRespRaw := fmt.Sprintf(`
		{
			"functions": [%s, %s]
		}
		`,
		mockContainerBasedFunctionInfo,
		mockHelmBasedFunctionInfo)
	var listNvidiaCloudFunctionVersionsMockResp ListNvidiaCloudFunctionVersionsResponse
	json.Unmarshal([]byte(listNvidiaCloudFunctionVersionsMockRespRaw), &listNvidiaCloudFunctionVersionsMockResp)

	type fields struct {
		NgcEndpoint string
		NgcApiKey   string
		NgcOrg      string
		NgcTeam     string
		HttpClient  *http.Client
	}
	type args struct {
		ctx        context.Context
		functionID string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantResp *ListNvidiaCloudFunctionVersionsResponse
		wantErr  bool
	}{
		{
			name: "ListNvidiaCloudFunctionVersions",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/functions/%s/versions", mockEndpoint, mockOrg, mockTeam, mockFunctionID),
						http.MethodGet,
						nvcfRequestHeaders,
						nil,
						listNvidiaCloudFunctionVersionsMockRespRaw,
						200,
					),
				},
			},
			args: args{
				ctx:        context.Background(),
				functionID: mockFunctionID,
			},
			wantResp: &listNvidiaCloudFunctionVersionsMockResp,
			wantErr:  false,
		},
		{
			name: "ListNvidiaCloudFunctionVersionsFailed",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/functions/%s/versions", mockEndpoint, mockOrg, mockTeam, mockFunctionID),
						http.MethodGet,
						nvcfRequestHeaders,
						nil,
						listNvidiaCloudFunctionVersionsMockRespRaw,
						500,
					),
				},
			},
			args: args{
				ctx:        context.Background(),
				functionID: mockFunctionID,
			},
			wantResp: &ListNvidiaCloudFunctionVersionsResponse{},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &NVCFClient{
				NgcEndpoint: tt.fields.NgcEndpoint,
				NgcApiKey:   tt.fields.NgcApiKey,
				NgcOrg:      tt.fields.NgcOrg,
				NgcTeam:     tt.fields.NgcTeam,
				HttpClient:  tt.fields.HttpClient,
			}
			gotResp, err := c.ListNvidiaCloudFunctionVersions(tt.args.ctx, tt.args.functionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NVCFClient.ListNvidiaCloudFunctionVersions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResp, tt.wantResp) {
				t.Errorf("NVCFClient.ListNvidiaCloudFunctionVersions() = %v, want %v", gotResp, tt.wantResp)
			}
		})
	}
}

func TestNVCFClient_DeleteNvidiaCloudFunctionVersion(t *testing.T) {
	t.Parallel()

	type fields struct {
		NgcEndpoint string
		NgcApiKey   string
		NgcOrg      string
		NgcTeam     string
		HttpClient  *http.Client
	}
	type args struct {
		ctx               context.Context
		functionID        string
		functionVersionID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "DeleteNvidiaCloudFunctionVersion",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodDelete,
						nvcfRequestHeaders,
						nil,
						"",
						204,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
			},
			wantErr: false,
		},
		{
			name: "DeleteNvidiaCloudFunctionVersionFailed",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodDelete,
						nvcfRequestHeaders,
						nil,
						"",
						500,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &NVCFClient{
				NgcEndpoint: tt.fields.NgcEndpoint,
				NgcApiKey:   tt.fields.NgcApiKey,
				NgcOrg:      tt.fields.NgcOrg,
				NgcTeam:     tt.fields.NgcTeam,
				HttpClient:  tt.fields.HttpClient,
			}
			if err := c.DeleteNvidiaCloudFunctionVersion(tt.args.ctx, tt.args.functionID, tt.args.functionVersionID); (err != nil) != tt.wantErr {
				t.Errorf("NVCFClient.DeleteNvidiaCloudFunctionVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNVCFClient_CreateNvidiaCloudFunctionDeployment(t *testing.T) {
	t.Parallel()

	var createNvidiaCloudFunctionDeploymentReq CreateNvidiaCloudFunctionDeploymentRequest
	var createNvidiaCloudFunctionDeploymentReqRaw = fmt.Sprintf(
		`{"deploymentSpecifications": [%s]}`,
		mockDeploymentSpecification,
	)
	json.Unmarshal([]byte(createNvidiaCloudFunctionDeploymentReqRaw), &createNvidiaCloudFunctionDeploymentReq)

	var createNvidiaCloudFunctionDeploymentResp CreateNvidiaCloudFunctionDeploymentResponse
	json.Unmarshal([]byte(mockFunctionDeploymentInfo), &createNvidiaCloudFunctionDeploymentResp)

	type fields struct {
		NgcEndpoint string
		NgcApiKey   string
		NgcOrg      string
		NgcTeam     string
		HttpClient  *http.Client
	}
	type args struct {
		ctx               context.Context
		functionID        string
		functionVersionID string
		req               CreateNvidiaCloudFunctionDeploymentRequest
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantResp *CreateNvidiaCloudFunctionDeploymentResponse
		wantErr  bool
	}{
		{
			name: "CreateNvidiaCloudFunctionDeployment",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/deployments/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodPost,
						nvcfRequestHeaders,
						createNvidiaCloudFunctionDeploymentReq,
						mockFunctionDeploymentInfo,
						200,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
				req:               createNvidiaCloudFunctionDeploymentReq,
			},
			wantResp: &createNvidiaCloudFunctionDeploymentResp,
			wantErr:  false,
		},
		{
			name: "CreateNvidiaCloudFunctionDeploymentFailed",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/deployments/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodPost,
						nvcfRequestHeaders,
						createNvidiaCloudFunctionDeploymentReq,
						mockFunctionDeploymentInfo,
						500,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
				req:               createNvidiaCloudFunctionDeploymentReq,
			},
			wantResp: &CreateNvidiaCloudFunctionDeploymentResponse{},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &NVCFClient{
				NgcEndpoint: tt.fields.NgcEndpoint,
				NgcApiKey:   tt.fields.NgcApiKey,
				NgcOrg:      tt.fields.NgcOrg,
				NgcTeam:     tt.fields.NgcTeam,
				HttpClient:  tt.fields.HttpClient,
			}
			gotResp, err := c.CreateNvidiaCloudFunctionDeployment(tt.args.ctx, tt.args.functionID, tt.args.functionVersionID, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NVCFClient.CreateNvidiaCloudFunctionDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResp, tt.wantResp) {
				t.Errorf("NVCFClient.CreateNvidiaCloudFunctionDeployment() = %v, want %v", gotResp, tt.wantResp)
			}
		})
	}
}

func TestNVCFClient_UpdateNvidiaCloudFunctionDeployment(t *testing.T) {
	t.Parallel()

	var updateNvidiaCloudFunctionDeploymentReq UpdateNvidiaCloudFunctionDeploymentRequest
	var updateNvidiaCloudFunctionDeploymentReqRaw = fmt.Sprintf(
		`{"deploymentSpecifications": [%s]}`,
		mockDeploymentSpecification,
	)
	json.Unmarshal([]byte(updateNvidiaCloudFunctionDeploymentReqRaw), &updateNvidiaCloudFunctionDeploymentReq)

	var updateNvidiaCloudFunctionDeploymentResp UpdateNvidiaCloudFunctionDeploymentResponse
	json.Unmarshal([]byte(mockFunctionDeploymentInfo), &updateNvidiaCloudFunctionDeploymentResp)

	type fields struct {
		NgcEndpoint string
		NgcApiKey   string
		NgcOrg      string
		NgcTeam     string
		HttpClient  *http.Client
	}
	type args struct {
		ctx               context.Context
		functionID        string
		functionVersionID string
		req               UpdateNvidiaCloudFunctionDeploymentRequest
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantResp *UpdateNvidiaCloudFunctionDeploymentResponse
		wantErr  bool
	}{
		{
			name: "UpdateNvidiaCloudFunctionDeployment",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/deployments/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodPut,
						nvcfRequestHeaders,
						updateNvidiaCloudFunctionDeploymentReq,
						mockFunctionDeploymentInfo,
						200,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
				req:               updateNvidiaCloudFunctionDeploymentReq,
			},
			wantResp: &updateNvidiaCloudFunctionDeploymentResp,
			wantErr:  false,
		},
		{
			name: "UpdateNvidiaCloudFunctionDeploymentFailed",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/deployments/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodPut,
						nvcfRequestHeaders,
						updateNvidiaCloudFunctionDeploymentReq,
						mockFunctionDeploymentInfo,
						500,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
				req:               updateNvidiaCloudFunctionDeploymentReq,
			},
			wantResp: &UpdateNvidiaCloudFunctionDeploymentResponse{},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &NVCFClient{
				NgcEndpoint: tt.fields.NgcEndpoint,
				NgcApiKey:   tt.fields.NgcApiKey,
				NgcOrg:      tt.fields.NgcOrg,
				NgcTeam:     tt.fields.NgcTeam,
				HttpClient:  tt.fields.HttpClient,
			}
			gotResp, err := c.UpdateNvidiaCloudFunctionDeployment(tt.args.ctx, tt.args.functionID, tt.args.functionVersionID, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NVCFClient.UpdateNvidiaCloudFunctionDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResp, tt.wantResp) {
				t.Errorf("NVCFClient.UpdateNvidiaCloudFunctionDeployment() = %v, want %v", gotResp, tt.wantResp)
			}
		})
	}
}

func TestNVCFClient_WaitingDeploymentCompleted(t *testing.T) {
	t.Parallel()

	var readNvidiaCloudFunctionDeploymentResp ReadNvidiaCloudFunctionDeploymentResponse
	json.Unmarshal([]byte(mockFunctionDeploymentInfo), &readNvidiaCloudFunctionDeploymentResp)

	type fields struct {
		NgcEndpoint string
		NgcApiKey   string
		NgcOrg      string
		NgcTeam     string
		HttpClient  *http.Client
	}
	type args struct {
		ctx               context.Context
		functionID        string
		functionVersionID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "WaitingDeploymentCompleted",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/deployments/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodGet,
						nvcfRequestHeaders,
						nil,
						mockFunctionDeploymentActiveInfo,
						200,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
			},
			wantErr: false,
		},
		{
			name: "WaitingDeploymentCompletedFailedWithStatusCode",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/deployments/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodGet,
						nvcfRequestHeaders,
						nil,
						mockFunctionDeploymentInfo,
						500,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
			},
			wantErr: true,
		},
		{
			name: "WaitingDeploymentCompletedFailed",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/deployments/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodGet,
						nvcfRequestHeaders,
						nil,
						mockFunctionDeploymentFailedInfo,
						200,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &NVCFClient{
				NgcEndpoint: tt.fields.NgcEndpoint,
				NgcApiKey:   tt.fields.NgcApiKey,
				NgcOrg:      tt.fields.NgcOrg,
				NgcTeam:     tt.fields.NgcTeam,
				HttpClient:  tt.fields.HttpClient,
			}
			if err := c.WaitingDeploymentCompleted(tt.args.ctx, tt.args.functionID, tt.args.functionVersionID); (err != nil) != tt.wantErr {
				t.Errorf("NVCFClient.WaitingDeploymentCompleted() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNVCFClient_ReadNvidiaCloudFunctionDeployment(t *testing.T) {
	t.Parallel()

	var updateNvidiaCloudFunctionDeploymentResp ReadNvidiaCloudFunctionDeploymentResponse
	json.Unmarshal([]byte(mockFunctionDeploymentInfo), &updateNvidiaCloudFunctionDeploymentResp)

	type fields struct {
		NgcEndpoint string
		NgcApiKey   string
		NgcOrg      string
		NgcTeam     string
		HttpClient  *http.Client
	}
	type args struct {
		ctx               context.Context
		functionID        string
		functionVersionID string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantResp *ReadNvidiaCloudFunctionDeploymentResponse
		wantErr  bool
	}{
		{
			name: "ReadNvidiaCloudFunctionDeployment",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/deployments/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodGet,
						nvcfRequestHeaders,
						nil,
						mockFunctionDeploymentInfo,
						200,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
			},
			wantResp: &updateNvidiaCloudFunctionDeploymentResp,
			wantErr:  false,
		},
		{
			name: "ReadNvidiaCloudFunctionDeploymentFailed",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/deployments/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodGet,
						nvcfRequestHeaders,
						nil,
						mockFunctionDeploymentInfo,
						500,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
			},
			wantResp: &ReadNvidiaCloudFunctionDeploymentResponse{},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &NVCFClient{
				NgcEndpoint: tt.fields.NgcEndpoint,
				NgcApiKey:   tt.fields.NgcApiKey,
				NgcOrg:      tt.fields.NgcOrg,
				NgcTeam:     tt.fields.NgcTeam,
				HttpClient:  tt.fields.HttpClient,
			}
			gotResp, err := c.ReadNvidiaCloudFunctionDeployment(tt.args.ctx, tt.args.functionID, tt.args.functionVersionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NVCFClient.ReadNvidiaCloudFunctionDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResp, tt.wantResp) {
				t.Errorf("NVCFClient.ReadNvidiaCloudFunctionDeployment() = %v, want %v", gotResp, tt.wantResp)
			}
		})
	}
}

func TestNVCFClient_DeleteNvidiaCloudFunctionDeployment(t *testing.T) {
	t.Parallel()

	deleteNvidiaCloudFunctionDeploymentMockRespRaw := mockHelmBasedFunctionInfo
	var deleteNvidiaCloudFunctionDeploymentMockResp DeleteNvidiaCloudFunctionDeploymentResponse
	json.Unmarshal([]byte(deleteNvidiaCloudFunctionDeploymentMockRespRaw), &deleteNvidiaCloudFunctionDeploymentMockResp)

	type fields struct {
		NgcEndpoint string
		NgcApiKey   string
		NgcOrg      string
		NgcTeam     string
		HttpClient  *http.Client
	}
	type args struct {
		ctx               context.Context
		functionID        string
		functionVersionID string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantResp *DeleteNvidiaCloudFunctionDeploymentResponse
		wantErr  bool
	}{
		{
			name: "DeleteNvidiaCloudFunctionDeployment",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/deployments/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodDelete,
						nvcfRequestHeaders,
						nil,
						mockFunctionDeploymentInfo,
						200,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
			},
			wantResp: &deleteNvidiaCloudFunctionDeploymentMockResp,
			wantErr:  false,
		},
		{
			name: "DeleteNvidiaCloudFunctionDeploymentFailed",
			fields: fields{
				NgcEndpoint: mockEndpoint,
				NgcApiKey:   mockApiKey,
				NgcOrg:      mockOrg,
				NgcTeam:     mockTeam,
				HttpClient: &http.Client{
					Transport: GenerateHttpClientMockRoundTripper(
						t,
						fmt.Sprintf("%s/v2/orgs/%s/teams/%s/nvcf/deployments/functions/%s/versions/%s", mockEndpoint, mockOrg, mockTeam, mockFunctionID, mockVersionID),
						http.MethodDelete,
						nvcfRequestHeaders,
						nil,
						mockFunctionDeploymentInfo,
						500,
					),
				},
			},
			args: args{
				ctx:               context.Background(),
				functionID:        mockFunctionID,
				functionVersionID: mockVersionID,
			},
			wantResp: &DeleteNvidiaCloudFunctionDeploymentResponse{},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &NVCFClient{
				NgcEndpoint: tt.fields.NgcEndpoint,
				NgcApiKey:   tt.fields.NgcApiKey,
				NgcOrg:      tt.fields.NgcOrg,
				NgcTeam:     tt.fields.NgcTeam,
				HttpClient:  tt.fields.HttpClient,
			}
			gotResp, err := c.DeleteNvidiaCloudFunctionDeployment(tt.args.ctx, tt.args.functionID, tt.args.functionVersionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NVCFClient.DeleteNvidiaCloudFunctionDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResp, tt.wantResp) {
				t.Errorf("NVCFClient.DeleteNvidiaCloudFunctionDeployment() = %v, want %v", gotResp, tt.wantResp)
			}
		})
	}
}
