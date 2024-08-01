//go:build !unittest
// +build !unittest

package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab-master.nvidia.com/nvb/core/terraform-provider-ngc/internal/provider/testutils"
)

func generateStateResourceId(resourceName string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		var rawState map[string]string
		for _, m := range state.Modules {
			if len(m.Resources) > 0 {
				if v, ok := m.Resources[resourceName]; ok {
					rawState = v.Primary.Attributes
				}
			}
		}
		return fmt.Sprintf("%s,%s", rawState["id"], rawState["version_id"]), nil
	}
}

func TestAccCloudFunctionResource_HelmBasedFunction(t *testing.T) {
	var functionName = uuid.New().String()
	var testCloudFunctionResourceName = fmt.Sprintf("terraform-cloud-function-integ-resource-%s", functionName)
	var testCloudFunctionResourceFullPath = fmt.Sprintf("ngc_cloud_function.%s", testCloudFunctionResourceName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Verify Function Creation Timeout
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
						    function_name           = "%s"
							helm_chart              = "%s"
							helm_chart_service_name = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health_uri              = "%s"
							api_body_format         = "%s"
							deployment_specifications = [
								{
									configuration           = "%s"
									backend                 = "%s"
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 1
									min_instances           = 1
									max_request_concurrency = 1
								}
							]
							timeouts = {
								create = "1s"
							}
						}
						`,
					testCloudFunctionResourceName,
					functionName,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWrite),
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				ExpectError: regexp.MustCompile("timeout occurred"),
			},
			// Verify Function Creation with NVCF API error
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
						    function_name           = "%s"
							helm_chart              = "%s"
							helm_chart_service_name = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health_uri              = "%s"
							api_body_format         = "%s"
							deployment_specifications = [
								{
									configuration           = "%s"
									backend                 = "%s"
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 1
									min_instances           = 2
									max_request_concurrency = 1
								}
							]
							timeouts = {
								create = "1s"
							}
						}
						`,
					testCloudFunctionResourceName,
					functionName,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWrite),
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				ExpectError: regexp.MustCompile("Validation failure"),
			},
			// Verify Function Creation
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
						    function_name             = "%s"
							helm_chart                = "%s"
							helm_chart_service_name   = "%s"
							inference_port            = %d
							inference_url             = "%s"
							health_uri                = "%s"
							api_body_format           = "%s"
							deployment_specifications = [
								{
									configuration           = "%s"
									backend                 = "%s"
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 1
									min_instances           = 1
									max_request_concurrency = 1
								}
							]
							tags = ["%s","%s"]
						}
						`,
					testCloudFunctionResourceName,
					functionName,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWrite),
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
					testutils.TestTags[0],
					testutils.TestTags[1],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "id"),
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "function_id"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "container_image"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart", testutils.TestHelmUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name", testutils.TestHelmServiceName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestHelmInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health_uri", testutils.TestHelmHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestHelmAPIFormat),
					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration", testutils.TestHelmValueOverWrite),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "tags.0", testutils.TestTags[0]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "tags.1", testutils.TestTags[1]),
				),
			},
			// Verify Function Update Timeout
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
						    function_name           = "%s"
							helm_chart              = "%s"
							helm_chart_service_name = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health_uri              = "%s"
							api_body_format         = "%s"
							deployment_specifications = [
								{
									configuration           = "%s"
									backend                 = "%s"
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 2
									min_instances           = 1
									max_request_concurrency = 1
								}
							]
							timeouts = {
								update = "1s"
							}
						}
						`,
					testCloudFunctionResourceName,
					functionName,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWrite),
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				ExpectError: regexp.MustCompile("timeout occurred"),
			},
			// Verify Function Update
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
						    function_name             = "%s"
							helm_chart                = "%s"
							helm_chart_service_name   = "%s"
							inference_port            = %d
							inference_url             = "%s"
							health_uri                = "%s"
							api_body_format           = "%s"
							deployment_specifications = [
								{
									configuration           = "%s"
									backend                 = "%s"
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 2
									min_instances           = 1
									max_request_concurrency = 2
								}
							]
						}
						`,
					testCloudFunctionResourceName,
					functionName,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWrite),
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "id"),
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "function_id"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "container_image"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart", testutils.TestHelmUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name", testutils.TestHelmServiceName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestHelmInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health_uri", testutils.TestHelmHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestHelmAPIFormat),
					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration", testutils.TestHelmValueOverWrite),
				),
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"keep_failed_resource", // Not assigned when import
				},
			},
		},
	})
}

func TestAccCloudFunctionResource_HelmBasedFunctionVersion(t *testing.T) {
	var functionName = uuid.New().String()
	var testCloudFunctionResourceName = fmt.Sprintf("terraform-cloud-function-integ-resource-%s", functionName)
	var testCloudFunctionResourceFullPath = fmt.Sprintf("ngc_cloud_function.%s", testCloudFunctionResourceName)

	functionInfo := testutils.CreateHelmFunction(t)
	defer testutils.DeleteFunction(t, functionInfo.Function.ID, functionInfo.Function.VersionID)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Verify Function Creation
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
							function_name           = "%s"
						    function_id             = "%s"
							helm_chart              = "%s"
							helm_chart_service_name = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health_uri              = "%s"
							api_body_format         = "%s"
							deployment_specifications = [
								{
									configuration           = "%s"
									backend                 = "%s"
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 1
									min_instances           = 1
									max_request_concurrency = 1
								}
							]
						}
						`,
					testCloudFunctionResourceName,
					functionName,
					functionInfo.Function.ID,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWrite),
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify version ID exist
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					// Verify container attribute not exist
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "container_image"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart", testutils.TestHelmUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name", testutils.TestHelmServiceName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestHelmInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health_uri", testutils.TestHelmHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestHelmAPIFormat),

					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration", testutils.TestHelmValueOverWrite),
				),
			},
			// Verify Function Update
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
						    function_name           = "%s"
						    function_id             = "%s"
							helm_chart              = "%s"
							helm_chart_service_name = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health_uri              = "%s"
							api_body_format         = "%s"
							deployment_specifications = [
								{
									configuration           = "%s"
									backend                 = "%s"
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 2
									min_instances           = 1
									max_request_concurrency = 2
								}
							]
						}
						`,
					testCloudFunctionResourceName,
					functionName,
					functionInfo.Function.ID,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWrite),
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify version ID exist
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					// Verify container attribute not exist
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "container_image"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart", testutils.TestHelmUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name", testutils.TestHelmServiceName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestHelmInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health_uri", testutils.TestHelmHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestHelmAPIFormat),
					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration", testutils.TestHelmValueOverWrite),
				),
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"function_id",          // Not assigned when import
					"keep_failed_resource", // Not assigned when import
				},
			},
		},
	})
}

func TestAccCloudFunctionResource_ContainerBasedFunction(t *testing.T) {
	var functionName = uuid.New().String()
	var testCloudFunctionResourceName = fmt.Sprintf("terraform-cloud-function-integ-resource-%s", functionName)
	var testCloudFunctionResourceFullPath = fmt.Sprintf("ngc_cloud_function.%s", testCloudFunctionResourceName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Verify Function Creation
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
						    function_name             = "%s"
							container_image           = "%s"
							inference_port            = %d
							inference_url             = "%s"
							health_uri                = "%s"
							api_body_format           = "%s"
							deployment_specifications = [
								{
									backend                 = "%s"
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 1
									min_instances           = 1
									max_request_concurrency = 1
								}
							]
							tags = ["%s","%s"]
							container_environment = [
								{
									key   = "%s"
									value = "%s"
								}
							]
						}
						`,
					testCloudFunctionResourceName,
					functionName,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerEndpoint,
					testutils.TestContainerHealthEndpoint,
					testutils.TestContainerAPIFormat,
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
					testutils.TestTags[0],
					testutils.TestTags[1],
					testutils.TestContainerEnvironmentVariables[0].Key,
					testutils.TestContainerEnvironmentVariables[0].Value,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "id"),
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "function_id"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerEndpoint),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health_uri", testutils.TestContainerHealthEndpoint),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),
					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "1"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "tags.0", testutils.TestTags[0]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "tags.1", testutils.TestTags[1]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_environment.0.key", testutils.TestContainerEnvironmentVariables[0].Key),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_environment.0.value", testutils.TestContainerEnvironmentVariables[0].Value),
				),
			},
			// Verify Function Update
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
						    function_name           = "%s"
							container_image         = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health_uri              = "%s"
							api_body_format         = "%s"
							deployment_specifications = [
								{
									backend                 = "%s"
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 2
									min_instances           = 1
									max_request_concurrency = 2
								}
							]
						}
						`,
					testCloudFunctionResourceName,
					functionName,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerEndpoint,
					testutils.TestContainerHealthEndpoint,
					testutils.TestContainerAPIFormat,
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "id"),
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "function_id"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerEndpoint),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health_uri", testutils.TestContainerHealthEndpoint),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "2"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration"),
				),
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"keep_failed_resource", // Not assigned when import
				},
			},
		},
	})
}

func TestAccCloudFunctionResource_ContainerBasedFunctionVersion(t *testing.T) {
	var functionName = uuid.New().String()
	var testCloudFunctionResourceName = fmt.Sprintf("terraform-cloud-function-integ-resource-%s", functionName)
	var testCloudFunctionResourceFullPath = fmt.Sprintf("ngc_cloud_function.%s", testCloudFunctionResourceName)

	functionInfo := testutils.CreateContainerFunction(t)
	defer testutils.DeleteFunction(t, functionInfo.Function.ID, functionInfo.Function.VersionID)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Verify Function Creation
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
							function_name           = "%s"
						    function_id             = "%s"
							container_image         = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health_uri              = "%s"
							api_body_format         = "%s"
							deployment_specifications = [
								{
									backend                 = "%s"
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 1
									min_instances           = 1
									max_request_concurrency = 1
								}
							]
						}
						`,
					testCloudFunctionResourceName,
					functionName,
					functionInfo.Function.ID,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerEndpoint,
					testutils.TestContainerHealthEndpoint,
					testutils.TestContainerAPIFormat,
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerEndpoint),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health_uri", testutils.TestContainerHealthEndpoint),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "1"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration"),
				),
			},
			// Verify Function Update
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
							function_name           = "%s"
						    function_id             = "%s"
							container_image         = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health_uri              = "%s"
							api_body_format         = "%s"
							deployment_specifications = [
								{
									backend                 = "%s"
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 2
									min_instances           = 1
									max_request_concurrency = 2
								}
							]
						}
						`,
					testCloudFunctionResourceName,
					functionName,
					functionInfo.Function.ID,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerEndpoint,
					testutils.TestContainerHealthEndpoint,
					testutils.TestContainerAPIFormat,
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerEndpoint),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health_uri", testutils.TestContainerHealthEndpoint),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "2"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration"),
				),
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"function_id",          // Not assigned when import
					"keep_failed_resource", // Not assigned when import
				},
			},
		},
	})
}
