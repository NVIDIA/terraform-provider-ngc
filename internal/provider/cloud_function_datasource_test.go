//  SPDX-FileCopyrightText: Copyright (c) 2024 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
//  SPDX-License-Identifier: LicenseRef-NvidiaProprietary

//  NVIDIA CORPORATION, its affiliates and licensors retain all intellectual
//  property and proprietary rights in and to this material, related
//  documentation and any modifications thereto. Any use, reproduction,
//  disclosure or distribution of this material and related documentation
//  without an express license agreement from NVIDIA CORPORATION or
//  its affiliates is strictly prohibited.

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
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "helm_chart", testutils.TestHelmUri),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "helm_chart_service_name", testutils.TestHelmServiceName),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "inference_port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "inference_url", testutils.TestHelmInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "health_uri", testutils.TestHelmHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "api_body_format", testutils.TestHelmAPIFormat),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckNoResourceAttr(testCloudFunctionDatasourceFullPath, "container_image"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.max_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.max_request_concurrency", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.configuration", testutils.TestHelmValueOverWrite),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "tags.0", testutils.TestTags[0]),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "tags.1", testutils.TestTags[1]),
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
					resource.TestCheckNoResourceAttr(testCloudFunctionDatasourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionDatasourceFullPath, "helm_chart_service_name"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "health_uri", testutils.TestContainerHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.max_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.max_request_concurrency", "1"),
					resource.TestCheckNoResourceAttr(testCloudFunctionDatasourceFullPath, "deployment_specifications.0.configuration"),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "tags.0", testutils.TestTags[0]),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "tags.1", testutils.TestTags[1]),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "container_environment.0.key", testutils.TestContainerEnvironmentVariables[0].Key),
					resource.TestCheckResourceAttr(testCloudFunctionDatasourceFullPath, "container_environment.0.value", testutils.TestContainerEnvironmentVariables[0].Value),
				),
			},
		},
	})
}
