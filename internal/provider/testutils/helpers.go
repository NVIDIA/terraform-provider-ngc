// Copyright 2024 NVIDIA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

var TestTags []string

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

	TestTags = []string{"mock1", "mock2"}
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
		t.Fatalf(fmt.Sprintf("Unable to create function: %s", err.Error()))
	}

	return resp
}

func CreateDeployment(t *testing.T, functionID string, versionID string, configurationRaw string) *utils.CreateNvidiaCloudFunctionDeploymentResponse {
	t.Helper()

	var configuration interface{}
	if configurationRaw != "" {
		err := json.Unmarshal([]byte(configurationRaw), &configuration)
		if err != nil {
			t.Fatalf(fmt.Sprintf("Unable to parse configurationRaw: %s", err.Error()))
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
		t.Fatalf(fmt.Sprintf("Unable to create function deployment: %s", err.Error()))
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
		t.Fatalf(fmt.Sprintf("Unable to create function: %s", err.Error()))
	}

	return resp
}

func DeleteFunction(t *testing.T, functionID string, versionID string) {
	t.Helper()

	err := TestNVCFClient.DeleteNvidiaCloudFunctionVersion(Ctx, functionID, versionID)

	if err != nil {
		t.Fatalf(fmt.Sprintf("Unable to delete function: %s", err.Error()))
	}
}

func EscapeJSON(t *testing.T, rawJson string) string {
	return strings.ReplaceAll(rawJson, "\"", "\\\"")
}
