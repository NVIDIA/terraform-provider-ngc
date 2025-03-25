//  SPDX-FileCopyrightText: Copyright (c) 2024 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
//  SPDX-License-Identifier: LicenseRef-NvidiaProprietary

//  NVIDIA CORPORATION, its affiliates and licensors retain all intellectual
//  property and proprietary rights in and to this material, related
//  documentation and any modifications thereto. Any use, reproduction,
//  disclosure or distribution of this material and related documentation
//  without an express license agreement from NVIDIA CORPORATION or
//  its affiliates is strictly prohibited.

package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type NVCFClient struct {
	NgcEndpoint string
	NgcApiKey   string
	NgcOrg      string
	NgcTeam     string
	HttpClient  *http.Client
}

func (c *NVCFClient) NvcfEndpoint(context.Context) string {
	if c.NgcTeam == "" {
		return fmt.Sprintf("%s/v2/orgs/%s", c.NgcEndpoint, c.NgcOrg)
	} else {
		return fmt.Sprintf("%s/v2/orgs/%s/teams/%s", c.NgcEndpoint, c.NgcOrg, c.NgcTeam)
	}
}

func (c *NVCFClient) HTTPClient(context.Context) *http.Client {
	return c.HttpClient
}

func (c *NVCFClient) sendRequest(ctx context.Context, requestURL string, method string, requestBody any, responseObject any, expectedStatusCode map[int]bool) error {
	var request *http.Request

	if requestBody != nil {
		payloadBuf := new(bytes.Buffer)
		err := json.NewEncoder(payloadBuf).Encode(requestBody)
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("failed to parse request body %s", requestBody))
			return err
		}
		request, _ = http.NewRequest(method, requestURL, payloadBuf)
	} else {
		request, _ = http.NewRequest(method, requestURL, http.NoBody)
	}

	request.Header.Set("Authorization", "Bearer "+c.NgcApiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := c.HttpClient.Do(request)

	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("failed to send request to %s with method %s", requestURL, method))
		return err
	}

	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)

	ctx = tflog.SetField(ctx, "response_status", response.Status)
	ctx = tflog.SetField(ctx, "response_header", response.Header)
	ctx = tflog.SetField(ctx, "response_body", string(body))
	ctx = tflog.SetField(ctx, "request_body", requestBody)

	tflog.Debug(ctx, "Send request")

	if _, ok := expectedStatusCode[response.StatusCode]; !ok {
		tflog.Error(ctx, "got unexpected response code")

		// The unauthenticated response format is different with others
		if response.StatusCode == 401 {
			tflog.Error(ctx, "unauthenticated error")
			return errors.New("not authenticated")
		}

		var errResponseObject = &ErrorResponse{}
		err = json.Unmarshal(body, errResponseObject)

		if err != nil {
			ctx = tflog.SetField(ctx, "response_body", string(body))
			tflog.Error(ctx, "failed to parse error response body")
			return fmt.Errorf("failed to parse error response body. Response body: %s", string(body))
		}

		if errResponseObject.RequestStatus.StatusDescription != "" {
			return errors.New(errResponseObject.RequestStatus.StatusDescription)
		} else {
			return errors.New(errResponseObject.Detail)
		}
	}

	if responseObject != nil {
		err = json.Unmarshal(body, responseObject)

		if err != nil {
			tflog.Error(ctx, "failed to parse response body")
			return err
		}
	}

	return err
}

// Function Management APIs.
func (c *NVCFClient) CreateNvidiaCloudFunction(ctx context.Context, functionID string, req CreateNvidiaCloudFunctionRequest) (resp *CreateNvidiaCloudFunctionResponse, err error) {
	var createNvidiaCloudFunctionResponse CreateNvidiaCloudFunctionResponse

	var requestURL string
	if functionID != "" {
		requestURL = fmt.Sprintf("%s/nvcf/functions/%s/versions", c.NvcfEndpoint(ctx), functionID)
	} else {
		requestURL = fmt.Sprintf("%s/nvcf/functions", c.NvcfEndpoint(ctx))
	}

	err = c.sendRequest(ctx, requestURL, http.MethodPost, req, &createNvidiaCloudFunctionResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Create NVCF Function.")
	return &createNvidiaCloudFunctionResponse, err
}

func (c *NVCFClient) ListNvidiaCloudFunctionVersions(ctx context.Context, functionID string) (resp *ListNvidiaCloudFunctionVersionsResponse, err error) {
	var listNvidiaCloudFunctionVersionsResponse ListNvidiaCloudFunctionVersionsResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/functions/" + functionID + "/versions"

	err = c.sendRequest(ctx, requestURL, http.MethodGet, nil, &listNvidiaCloudFunctionVersionsResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "List NVCF Function versions")
	return &listNvidiaCloudFunctionVersionsResponse, err
}

func (c *NVCFClient) UpdateNvidiaCloudFunctionMetadata(ctx context.Context, functionID string, functionVersionID string, req UpdateNvidiaCloudFunctionMetadataRequest) (resp *UpdateNvidiaCloudFunctionMetadataResponse, err error) {
	var updateNvidiaCloudFunctionMetadataResponse UpdateNvidiaCloudFunctionMetadataResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/metadata/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodPut, req, &updateNvidiaCloudFunctionMetadataResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Update NVCF Function Metadata.")
	return &updateNvidiaCloudFunctionMetadataResponse, err
}

func (c *NVCFClient) GetNvidiaCloudFunctionVersion(ctx context.Context, functionID string, functionVersionID string) (resp *GetNvidiaCloudFunctionVersionResponse, err error) {
	var getNvidiaCloudFunctionVersionResponse GetNvidiaCloudFunctionVersionResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodGet, nil, &getNvidiaCloudFunctionVersionResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Get NVCF Function version")
	return &getNvidiaCloudFunctionVersionResponse, err
}

func (c *NVCFClient) DeleteNvidiaCloudFunctionVersion(ctx context.Context, functionID string, functionVersionID string) (err error) {
	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodDelete, nil, nil, map[int]bool{204: true})
	tflog.Debug(ctx, "Delete Function Deployment")
	return err
}

// Function Deployment APIs.
func (c *NVCFClient) CreateNvidiaCloudFunctionDeployment(ctx context.Context, functionID string, functionVersionID string, req CreateNvidiaCloudFunctionDeploymentRequest) (resp *CreateNvidiaCloudFunctionDeploymentResponse, err error) {
	var createNvidiaCloudFunctionDeploymentResponse CreateNvidiaCloudFunctionDeploymentResponse
	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/deployments/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodPost, req, &createNvidiaCloudFunctionDeploymentResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Create Function Deployment")
	return &createNvidiaCloudFunctionDeploymentResponse, err
}

func (c *NVCFClient) UpdateNvidiaCloudFunctionDeployment(ctx context.Context, functionID string, functionVersionID string, req UpdateNvidiaCloudFunctionDeploymentRequest) (resp *UpdateNvidiaCloudFunctionDeploymentResponse, err error) {
	var updateNvidiaCloudFunctionDeploymentResponse UpdateNvidiaCloudFunctionDeploymentResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/deployments/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodPut, req, &updateNvidiaCloudFunctionDeploymentResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Update Function Deployment")
	return &updateNvidiaCloudFunctionDeploymentResponse, err
}

func (c *NVCFClient) WaitingDeploymentCompleted(ctx context.Context, functionID string, functionVersionId string) error {
	for {
		readNvidiaCloudFunctionDeploymentResponse, err := c.ReadNvidiaCloudFunctionDeployment(ctx, functionID, functionVersionId)

		if err != nil {
			return err
		}

		if readNvidiaCloudFunctionDeploymentResponse.Deployment.FunctionStatus == "ACTIVE" {
			return nil
		} else if readNvidiaCloudFunctionDeploymentResponse.Deployment.FunctionStatus == "DEPLOYING" {
			select {
			case <-ctx.Done():
				return errors.New("timeout occurred")
			case <-time.After(60 * time.Second):
				continue
			}
		} else {
			return fmt.Errorf("unexpected status %s", readNvidiaCloudFunctionDeploymentResponse.Deployment.FunctionStatus)
		}
	}
}

func (c *NVCFClient) ReadNvidiaCloudFunctionDeployment(ctx context.Context, functionID string, functionVersionID string) (resp *ReadNvidiaCloudFunctionDeploymentResponse, err error) {
	var readNvidiaCloudFunctionDeploymentResponse ReadNvidiaCloudFunctionDeploymentResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/deployments/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodGet, nil, &readNvidiaCloudFunctionDeploymentResponse, map[int]bool{200: true, 404: true})
	tflog.Debug(ctx, "Read Function Deployment")
	return &readNvidiaCloudFunctionDeploymentResponse, err
}

func (c *NVCFClient) DeleteNvidiaCloudFunctionDeployment(ctx context.Context, functionID string, functionVersionID string) (resp *DeleteNvidiaCloudFunctionDeploymentResponse, err error) {
	var deleteNvidiaCloudFunctionDeploymentResponse DeleteNvidiaCloudFunctionDeploymentResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/deployments/functions/" + functionID + "/versions/" + functionVersionID
	err = c.sendRequest(ctx, requestURL, http.MethodDelete, nil, &deleteNvidiaCloudFunctionDeploymentResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Delete Function Deployment")
	return &deleteNvidiaCloudFunctionDeploymentResponse, err
}

// Function Sharing APIs.
func (c *NVCFClient) AuthorizeAccountsToInvokeFunction(ctx context.Context, functionID string, functionVersionID string, req AuthorizeAccountsToInvokeFunctionRequest) (resp *AuthorizeAccountsToInvokeFunctionResponse, err error) {
	var authorizeAccountsToInvokeFunctionResponse AuthorizeAccountsToInvokeFunctionResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/authorizations/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodPost, req, &authorizeAccountsToInvokeFunctionResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Authorize Accounts To Invoke Function")
	return &authorizeAccountsToInvokeFunctionResponse, err
}

func (c *NVCFClient) UnAuthorizeAllExtraAccountsToInvokeFunction(ctx context.Context, functionID string, functionVersionID string) (err error) {
	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/authorizations/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodDelete, nil, nil, map[int]bool{200: true})
	tflog.Debug(ctx, "Unauthorize All Extra Accounts To Invoke Function")
	return err
}

func (c *NVCFClient) GetFunctionAuthorization(ctx context.Context, functionID string, functionVersionID string) (resp *AuthorizeAccountsToInvokeFunctionResponse, err error) {
	var authorizeAccountsToInvokeFunctionResponse AuthorizeAccountsToInvokeFunctionResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/authorizations/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodGet, nil, &authorizeAccountsToInvokeFunctionResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Get Function Authorization")
	return &authorizeAccountsToInvokeFunctionResponse, err
}
