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

const RESOURCE_PREFIX = "terraform-provider-integ"

var TestNcaID string

var TestHelmFunctionName string
var TestHelmUri string
var TestHelmServiceName string
var TestHelmServicePort int
var TestHelmEndpointPath string
var TestHelmHealthEndpointPath string
var TestHelmValueOverWrite string
var TestHelmAPIFormat string

var TestContainerFunctionName string
var TestContainerUri string
var TestContainerPort int
var TestContainerEndpoint string
var TestContainerHealthEndpoint string
var TestContainerAPIFormat string

var TestBackend string
var TestInstanceType string
var TestGpuType string

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
	TestHelmFunctionName = fmt.Sprintf("%s-helm-function-01", RESOURCE_PREFIX)
	TestHelmUri = os.Getenv("HELM_CHART_URI")
	TestHelmServiceName = os.Getenv("HELM_CHART_SERVICE_NAME")
	TestHelmServicePort, _ = strconv.Atoi(os.Getenv("HELM_CHART_SERVICE_PORT"))
	TestHelmEndpointPath = os.Getenv("HELM_CHART_ENDPOINT_PATH")
	TestHelmHealthEndpointPath = os.Getenv("HELM_CHART_HEALTH_ENDPOINT_PATH")
	TestHelmValueOverWrite = os.Getenv("HELM_CHART_VALUE_YAML_OVERWRITE")
	TestHelmAPIFormat = "CUSTOM"

	// Container-Base Function
	TestContainerFunctionName = fmt.Sprintf("%s-container-function-01", RESOURCE_PREFIX)
	TestContainerUri = os.Getenv("CONTAINER_URI")
	TestContainerPort, _ = strconv.Atoi(os.Getenv("CONTAINER_PORT"))
	TestContainerEndpoint = os.Getenv("CONTAINER_ENDPOINT_PATH")
	TestContainerHealthEndpoint = os.Getenv("CONTAINER_HEALTH_ENDPOINT_PATH")
	TestContainerAPIFormat = "CUSTOM"

	TestBackend = os.Getenv("BACKEND")
	TestInstanceType = os.Getenv("INSTANCE_TYPE")
	TestGpuType = os.Getenv("GPU_TYPE")
}

func CreateHelmFunction(t *testing.T) *utils.CreateNvidiaCloudFunctionResponse {
	t.Helper()

	resp, err := TestNVCFClient.CreateNvidiaCloudFunction(Ctx, "", utils.CreateNvidiaCloudFunctionRequest{
		FunctionName:                       TestHelmFunctionName,
		HelmChartUri:                       TestHelmUri,
		HelmChartServiceName:               TestHelmServiceName,
		HelmChartServicePort:               TestHelmServicePort,
		HelmChartValuesOverwriteJsonString: TestHelmValueOverWrite,
		EndpointPath:                       TestHelmEndpointPath,
		HealthEndpointPath:                 TestHelmHealthEndpointPath,
		APIBodyFormat:                      TestHelmAPIFormat,
	})

	if err != nil {
		t.Fatalf(fmt.Sprintf("Unable to create function: %s", err.Error()))
	}

	return resp
}

func CreateDeployment(t *testing.T, functionID string, versionID string, configurationRaw string) *utils.CreateNvidiaCloudFunctionDeploymentResponse {
	t.Helper()

	var configuration interface{}
	json.Unmarshal([]byte(configurationRaw), &configuration)

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
		FunctionName:       TestContainerFunctionName,
		ContainerImageUri:  TestContainerUri,
		ContainerPort:      TestContainerPort,
		EndpointPath:       TestContainerEndpoint,
		HealthEndpointPath: TestContainerHealthEndpoint,
		APIBodyFormat:      TestContainerAPIFormat,
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

func EscapeJson(t *testing.T, rawJson string) string {
	return strings.ReplaceAll(rawJson, "\"", "\\\"")
}
