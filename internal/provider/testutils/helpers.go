//  SPDX-FileCopyrightText: Copyright (c) 2024 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
//  SPDX-License-Identifier: LicenseRef-NvidiaProprietary

//  NVIDIA CORPORATION, its affiliates and licensors retain all intellectual
//  property and proprietary rights in and to this material, related
//  documentation and any modifications thereto. Any use, reproduction,
//  disclosure or distribution of this material and related documentation
//  without an express license agreement from NVIDIA CORPORATION or
//  its affiliates is strictly prohibited.

package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/joho/godotenv"
	"gitlab-master.nvidia.com/nvb/core/terraform-provider-ngc/internal/provider/utils"
)

var TestNGCClient *utils.NGCClient
var TestNVCFClient *utils.NVCFClient
var Ctx = context.Background()

const resourcePrefix = "terraform-provider-integ"

var TestNcaID string
var TestFunctionType string

var TestHelmFunctionName string
var TestHelmUri string
var TestHelmServiceName string
var TestHelmServicePort int
var TestHelmInferenceUrl string
var TestHelmHealthUri string
var TestHelmValueOverWrite string
var TestHelmValueOverWriteUpdated string
var TestHelmAPIFormat string

var TestContainerFunctionName string
var TestContainerUri string
var TestContainerPort int
var TestContainerInferenceUrl string
var TestContainerHealthUri string
var TestContainerAPIFormat string
var TestContainerEnvironmentVariables []utils.NvidiaCloudFunctionContainerEnvironment

var TestBackend string
var TestInstanceType string
var TestGpuType string

var TestModel1Name string
var TestModel1Version string
var TestModel1Uri string

var TestTags []string
var TestSecretNames []string

var TestAuthorizedParty1 string
var TestAuthorizedParty2 string

func init() {
	err := godotenv.Load(os.Getenv("TEST_ENV_FILE"))

	if err != nil {
		log.Fatal("Error loading test config file")
	}

	TestNGCClient = &utils.NGCClient{
		NgcEndpoint: os.Getenv("NGC_ENDPOINT"),
		NgcApiKey:   os.Getenv("NGC_API_KEY"),
		NgcOrg:      os.Getenv("NGC_ORG"),
		NgcTeam:     os.Getenv("NGC_TEAM"),
		HttpClient:  cleanhttp.DefaultPooledClient(),
	}

	TestNcaID = os.Getenv("NCA_ID")
	TestNVCFClient = TestNGCClient.NVCFClient()

	// Setup Test Data

	// Helm-Base Function
	TestHelmFunctionName = fmt.Sprintf("%s-helm-function-01", resourcePrefix)
	TestHelmUri = os.Getenv("HELM_URI")
	TestHelmServiceName = os.Getenv("HELM_SERVICE_NAME")
	TestHelmServicePort, _ = strconv.Atoi(os.Getenv("HELM_SERVICE_PORT"))
	TestHelmInferenceUrl = os.Getenv("HELM_INFERENCE_URL")
	TestHelmHealthUri = os.Getenv("HELM_HEALTH_URI")
	TestHelmValueOverWrite = os.Getenv("HELM_VALUE_YAML_OVERWRITE")
	TestHelmValueOverWriteUpdated = os.Getenv("HELM_VALUE_YAML_OVERWRITE_UPDATE")
	TestHelmAPIFormat = "CUSTOM"

	// Container-Base Function
	TestContainerFunctionName = fmt.Sprintf("%s-container-function-01", resourcePrefix)
	TestContainerUri = os.Getenv("CONTAINER_URI")
	TestContainerPort, _ = strconv.Atoi(os.Getenv("CONTAINER_PORT"))
	TestContainerInferenceUrl = os.Getenv("CONTAINER_INFERENCE_URL")
	TestContainerHealthUri = os.Getenv("CONTAINER_HEALTH_URI")
	TestContainerAPIFormat = "CUSTOM"
	TestContainerEnvironmentVariables = []utils.NvidiaCloudFunctionContainerEnvironment{
		{
			Key:   "mock_key",
			Value: "mock_val",
		},
	}
	TestBackend = os.Getenv("BACKEND")
	TestInstanceType = os.Getenv("INSTANCE_TYPE")
	TestGpuType = os.Getenv("GPU_TYPE")
	TestFunctionType = "DEFAULT"

	TestModel1Name = os.Getenv("MODEL_1_NAME")
	TestModel1Version = os.Getenv("MODEL_1_VERSION")
	TestModel1Uri = os.Getenv("MODEL_1_URI")

	TestTags = []string{"mock1", "mock2"}
	TestSecretNames = []string{"1test-raw", "test-json", "test.s3.us-west-2.amazonaws.com"}

	TestAuthorizedParty1 = os.Getenv("AUTHORIZED_PARTY_1")
	TestAuthorizedParty2 = os.Getenv("AUTHORIZED_PARTY_2")
}

func CreateHelmFunction(t *testing.T) *utils.CreateNvidiaCloudFunctionResponse {
	t.Helper()

	resp, err := TestNVCFClient.CreateNvidiaCloudFunction(Ctx, "", utils.CreateNvidiaCloudFunctionRequest{
		FunctionName:         TestHelmFunctionName,
		HelmChart:            TestHelmUri,
		HelmChartServiceName: TestHelmServiceName,
		InferencePort:        TestHelmServicePort,
		InferenceUrl:         TestHelmInferenceUrl,
		HealthUri:            TestHelmHealthUri,
		APIBodyFormat:        TestHelmAPIFormat,
		Tags:                 TestTags,
		FunctionType:         TestFunctionType,
	})

	if err != nil {
		t.Fatalf("Unable to create function: %s", err.Error())
	}

	return resp
}

func CreateDeployment(t *testing.T, functionID string, versionID string, configurationRaw string) *utils.CreateNvidiaCloudFunctionDeploymentResponse {
	t.Helper()

	var configuration interface{}
	if configurationRaw != "" {
		err := json.Unmarshal([]byte(configurationRaw), &configuration)
		if err != nil {
			t.Fatalf("Unable to parse configurationRaw: %s", err.Error())
		}
	}

	resp, err := TestNVCFClient.CreateNvidiaCloudFunctionDeployment(Ctx, functionID, versionID, utils.CreateNvidiaCloudFunctionDeploymentRequest{
		DeploymentSpecifications: []utils.NvidiaCloudFunctionDeploymentSpecification{
			{
				Gpu:                   TestGpuType,
				Backend:               TestBackend,
				InstanceType:          TestInstanceType,
				MaxInstances:          1,
				MinInstances:          1,
				MaxRequestConcurrency: 1,
				Configuration:         configuration,
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to create function deployment: %s", err.Error())
	}

	return resp
}

func CreateContainerFunction(t *testing.T) *utils.CreateNvidiaCloudFunctionResponse {
	t.Helper()

	resp, err := TestNVCFClient.CreateNvidiaCloudFunction(Ctx, "", utils.CreateNvidiaCloudFunctionRequest{
		FunctionName:         TestContainerFunctionName,
		ContainerImage:       TestContainerUri,
		InferencePort:        TestContainerPort,
		InferenceUrl:         TestContainerInferenceUrl,
		HealthUri:            TestContainerHealthUri,
		APIBodyFormat:        TestContainerAPIFormat,
		Tags:                 TestTags,
		ContainerEnvironment: TestContainerEnvironmentVariables,
		FunctionType:         TestFunctionType,
	})

	if err != nil {
		t.Fatalf("Unable to create function: %s", err.Error())
	}

	return resp
}

func DeleteFunction(t *testing.T, functionID string, versionID string) {
	t.Helper()

	err := TestNVCFClient.DeleteNvidiaCloudFunctionVersion(Ctx, functionID, versionID)

	if err != nil {
		t.Fatalf("Unable to delete function: %s", err.Error())
	}
}

func EscapeJSON(t *testing.T, rawJson string) string {
	return strings.ReplaceAll(rawJson, "\"", "\\\"")
}
