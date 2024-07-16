//go:build !unittest
// +build !unittest

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab-master.nvidia.com/nvb/core/terraform-provider-ngc/internal/provider/testutils"
)

var testCloudFunctionDatasourceName = "terraform-cloud-function-integ-datasource"
var testCloudFunctionDatasourceFullPath = fmt.Sprintf("data.ngc_cloud_function.%s", testCloudFunctionDatasourceName)

func TestAccCloudFunctionDataSource_HelmBasedFunction(t *testing.T) {

	functionInfo := testutils.CreateHelmFunction(t)
	defer testutils.DeleteFunction(t, functionInfo.Function.ID, functionInfo.Function.VersionID)

	testutils.CreateDeployment(t, functionInfo.Function.ID, functionInfo.Function.VersionID, testutils.TestHelmValueOverWrite)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
						data "ngc_cloud_function" "%s" {
						function_id = "%s"
						version_id  = "%s"
						}
						`,
					testCloudFunctionDatasourceName, functionInfo.Function.ID, functionInfo.Function.VersionID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "function_id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "version_id", functionInfo.Function.VersionID),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "function_name", testutils.TestHelmFunctionName),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "helm_chart_uri", testutils.TestHelmUri),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "helm_chart_service_name", testutils.TestHelmServiceName),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "helm_chart_service_port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "endpoint_path", testutils.TestHelmEndpointPath),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "health_endpoint_path", testutils.TestHelmHealthEndpointPath),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "api_body_format", testutils.TestHelmAPIFormat),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckNoResourceAttr(testCloudFunctionDatasourceFullPath, "container_image_uri"),
					resource.TestCheckNoResourceAttr(testCloudFunctionDatasourceFullPath, "container_port"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.max_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.max_request_concurrency", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.configuration", testutils.TestHelmValueOverWrite),
				),
			},
		},
	})
}

func TestAccCloudFunctionDataSource_ContainerBasedFunction(t *testing.T) {

	functionInfo := testutils.CreateContainerFunction(t)
	defer testutils.DeleteFunction(t, functionInfo.Function.ID, functionInfo.Function.VersionID)

	testutils.CreateDeployment(t, functionInfo.Function.ID, functionInfo.Function.VersionID, "")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
						data "ngc_cloud_function" "%s" {
						function_id = "%s"
						version_id  = "%s"
						}
						`,
					testCloudFunctionDatasourceName, functionInfo.Function.ID, functionInfo.Function.VersionID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "function_id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "version_id", functionInfo.Function.VersionID),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "function_name", testutils.TestContainerFunctionName),
					resource.TestCheckNoResourceAttr(testCloudFunctionDatasourceFullPath, "helm_chart_uri"),
					resource.TestCheckNoResourceAttr(testCloudFunctionDatasourceFullPath, "helm_chart_service_name"),
					resource.TestCheckNoResourceAttr(testCloudFunctionDatasourceFullPath, "helm_chart_service_port"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "endpoint_path", testutils.TestContainerEndpoint),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "health_endpoint_path", testutils.TestContainerHealthEndpoint),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "container_image_uri", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "container_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.max_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.max_request_concurrency", "1"),
					resource.TestCheckNoResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.configuration"),
				),
			},
		},
	})
}
