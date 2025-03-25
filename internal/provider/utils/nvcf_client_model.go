//  SPDX-FileCopyrightText: Copyright (c) 2024 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
//  SPDX-License-Identifier: LicenseRef-NvidiaProprietary

package utils

import "time"

type RequestStatusModel struct {
	StatusCode        string `json:"statusCode"`
	StatusDescription string `json:"statusDescription"`
	RequestID         string `json:"requestId"`
}

type ErrorResponse struct {
	RequestStatus RequestStatusModel `json:"requestStatus"`
	// There are two format error response in NVCF endpoint.
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

type NvidiaCloudFunctionSecret struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

type NvidiaCloudFunctionModel struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URI     string `json:"uri"`
}

type NvidiaCloudFunctionContainerEnvironment struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type NvidiaCloudFunctionHealth struct {
	Protocol           string `json:"protocol,omitempty"`
	URI                string `json:"uri,omitempty"`
	Port               int    `json:"port,omitempty"`
	Timeout            string `json:"timeout,omitempty"`
	ExpectedStatusCode int    `json:"expectedStatusCode,omitempty"`
}

type NvidiaCloudFunctionActiveInstance struct {
	InstanceID        string    `json:"instanceId"`
	FunctionID        string    `json:"functionId"`
	FunctionVersionID string    `json:"functionVersionId"`
	InstanceType      string    `json:"instanceType"`
	InstanceStatus    string    `json:"instanceStatus"`
	SisRequestID      string    `json:"sisRequestId"`
	NcaID             string    `json:"ncaId"`
	Gpu               string    `json:"gpu"`
	Backend           string    `json:"backend"`
	Location          string    `json:"location"`
	InstanceCreatedAt time.Time `json:"instanceCreatedAt"`
	InstanceUpdatedAt time.Time `json:"instanceUpdatedAt"`
}

type NvidiaCloudFunctionResource struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URI     string `json:"uri"`
}

type NvidiaCloudFunctionInfo struct {
	ID                      string                                    `json:"id"`
	NcaID                   string                                    `json:"ncaId"`
	VersionID               string                                    `json:"versionId"`
	Name                    string                                    `json:"name"`
	Status                  string                                    `json:"status"`
	InferenceURL            string                                    `json:"inferenceUrl"`
	OwnedByDifferentAccount bool                                      `json:"ownedByDifferentAccount"`
	InferencePort           int                                       `json:"inferencePort"`
	ContainerImage          string                                    `json:"containerImage"`
	ContainerEnvironment    []NvidiaCloudFunctionContainerEnvironment `json:"containerEnvironment"`
	Models                  []NvidiaCloudFunctionModel                `json:"models"`
	ContainerArgs           string                                    `json:"containerArgs"`
	APIBodyFormat           string                                    `json:"apiBodyFormat"`
	HelmChart               string                                    `json:"helmChart"`
	HelmChartServiceName    string                                    `json:"helmChartServiceName"`
	HealthURI               string                                    `json:"healthUri"`
	CreatedAt               time.Time                                 `json:"createdAt"`
	Description             string                                    `json:"description"`
	Health                  *NvidiaCloudFunctionHealth                `json:"health"`
	ActiveInstances         []NvidiaCloudFunctionActiveInstance       `json:"activeInstances"`
	Resources               []NvidiaCloudFunctionResource             `json:"resources"`
	Secrets                 []string                                  `json:"secrets"`
	Tags                    []string                                  `json:"tags"`
	FunctionType            string                                    `json:"functionType"`
}

type CreateNvidiaCloudFunctionRequest struct {
	FunctionName         string                                    `json:"name"`
	HelmChart            string                                    `json:"helmChart,omitempty"`
	HelmChartServiceName string                                    `json:"helmChartServiceName,omitempty"`
	InferenceUrl         string                                    `json:"inferenceUrl"`
	HealthUri            string                                    `json:"healthUri,omitempty"`
	InferencePort        int                                       `json:"inferencePort"`
	ContainerImage       string                                    `json:"containerImage,omitempty"`
	ContainerEnvironment []NvidiaCloudFunctionContainerEnvironment `json:"containerEnvironment,omitempty"`
	Models               []NvidiaCloudFunctionModel                `json:"models,omitempty"`
	ContainerArgs        string                                    `json:"containerArgs,omitempty"`
	APIBodyFormat        string                                    `json:"apiBodyFormat"`
	Description          string                                    `json:"description,omitempty"`
	Health               *NvidiaCloudFunctionHealth                `json:"health,omitempty"`
	Resources            []NvidiaCloudFunctionResource             `json:"resources,omitempty"`
	Secrets              []NvidiaCloudFunctionSecret               `json:"secrets,omitempty"`
	Tags                 []string                                  `json:"tags,omitempty"`
	FunctionType         string                                    `json:"functionType"`
}

type CreateNvidiaCloudFunctionResponse struct {
	Function NvidiaCloudFunctionInfo `json:"function"`
}

type ListNvidiaCloudFunctionVersionsResponse struct {
	Functions []NvidiaCloudFunctionInfo `json:"functions"`
}

type ListNvidiaCloudFunctionVersionsRequest struct {
	FunctionID string `json:"name"`
}

type GetNvidiaCloudFunctionVersionResponse struct {
	Function NvidiaCloudFunctionInfo `json:"function"`
}

type UpdateNvidiaCloudFunctionMetadataRequest struct {
	Tags []string `json:"tags,omitempty"`
}

type UpdateNvidiaCloudFunctionMetadataResponse struct {
	Function NvidiaCloudFunctionInfo `json:"function"`
}

type NvidiaCloudFunctionDeploymentSpecification struct {
	Gpu                   string      `json:"gpu"`
	Backend               string      `json:"backend"`
	InstanceType          string      `json:"instanceType"`
	MaxInstances          int         `json:"maxInstances"`
	MinInstances          int         `json:"minInstances"`
	MaxRequestConcurrency int         `json:"maxRequestConcurrency"`
	Configuration         interface{} `json:"configuration"`
}

type NvidiaCloudFunctionDeployment struct {
	FunctionID               string                                       `json:"functionId"`
	FunctionVersionID        string                                       `json:"functionVersionId"`
	NcaID                    string                                       `json:"ncaId"`
	FunctionStatus           string                                       `json:"functionStatus"`
	HealthInfo               interface{}                                  `json:"healthInfo"`
	DeploymentSpecifications []NvidiaCloudFunctionDeploymentSpecification `json:"deploymentSpecifications"`
}

type CreateNvidiaCloudFunctionDeploymentRequest struct {
	DeploymentSpecifications []NvidiaCloudFunctionDeploymentSpecification `json:"deploymentSpecifications"`
}

type CreateNvidiaCloudFunctionDeploymentResponse struct {
	Deployment NvidiaCloudFunctionDeployment `json:"deployment"`
}

type UpdateNvidiaCloudFunctionDeploymentRequest struct {
	DeploymentSpecifications []NvidiaCloudFunctionDeploymentSpecification `json:"deploymentSpecifications"`
}

type UpdateNvidiaCloudFunctionDeploymentResponse struct {
	Deployment NvidiaCloudFunctionDeployment `json:"deployment"`
}

type ReadNvidiaCloudFunctionDeploymentResponse struct {
	Deployment NvidiaCloudFunctionDeployment `json:"deployment"`
}

type DeleteNvidiaCloudFunctionDeploymentResponse struct {
	Function NvidiaCloudFunctionInfo `json:"function"`
}

type AuthorizedParty struct {
	ClientId string `json:"clientId,omitempty"`
	NcaID    string `json:"ncaId"`
}

type AuthorizeAccountsToInvokeFunctionRequest struct {
	AuthorizedParties []AuthorizedParty `json:"authorizedParties"`
}

type AuthorizeAccountsToInvokeFunctionResponseFunctionInfo struct {
	Id                string            `json:"id"`
	NcaID             string            `json:"ncaId"`
	VersionID         string            `json:"versionId"`
	AuthorizedParties []AuthorizedParty `json:"authorizedParties"`
}

type AuthorizeAccountsToInvokeFunctionResponse struct {
	Function AuthorizeAccountsToInvokeFunctionResponseFunctionInfo `json:"function"`
}
