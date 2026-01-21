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
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab-master.nvidia.com/nvb/core/terraform-provider-ngc/internal/provider/testutils"
)

func generateFunctionStateResourceId(resourceName string) resource.ImportStateIdFunc {
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

func TestAccCloudFunctionResource_CreateHelmBasedFunctionFail(t *testing.T) {
	var functionName = testutils.TestCommonPrefix + "helm-based-function-fail"

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
							health                  = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
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
					functionName,
					functionName,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmServicePort,
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
							health                  = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
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
					functionName,
					functionName,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmServicePort,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWrite),
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				ExpectError: regexp.MustCompile("Validation failed with"),
			},
		},
	})
}

func TestAccCloudFunctionResource_CreateAndUpdateHelmBasedFunctionDeployWithBackendOptionSuccess(t *testing.T) {
	var functionName = testutils.TestCommonPrefix + "helm-based-function-with-backend-option"
	var testCloudFunctionResourceFullPath = fmt.Sprintf("ngc_cloud_function.%s", functionName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Verify Function Creation
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
						    function_name             = "%s"
							helm_chart                = "%s"
							helm_chart_service_name   = "%s"
							inference_port            = %d
							inference_url             = "%s"
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
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
							authorized_parties = [
								{
									nca_id = "%s"
								},
								{
									nca_id = "%s"
								}
							]
							tags = ["%s","%s"]
							graceful_deletion = true
						}
						`,
					functionName,
					functionName,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmServicePort,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWrite),
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
					testutils.TestAuthorizedParty1,
					testutils.TestAuthorizedParty2,
					testutils.TestTags[0],
					testutils.TestTags[1],
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "id"),
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "function_id"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "container_image"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "telemetries"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart", testutils.TestHelmUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name", testutils.TestHelmServiceName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestHelmInferenceUrl),
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

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestHelmHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "authorized_parties.#", "2"),
				),
			},
			// Verify Function In-place Update
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
						    function_name             = "%s"
							helm_chart                = "%s"
							helm_chart_service_name   = "%s"
							inference_port            = %d
							inference_url             = "%s"
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
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
							timeouts = {
								update = "3s" # The update will be returned quickly since it just trigger in-place update.
							}
						}
						`,
					functionName,
					functionName,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmServicePort,
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
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "telemetries"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart", testutils.TestHelmUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name", testutils.TestHelmServiceName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestHelmInferenceUrl),
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

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestHelmHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "authorized_parties.#", "0"),
				),
			},
			// Verify Function Force-Replace Update with Creation Timeout
			{
				Config: fmt.Sprintf(`
									resource "ngc_cloud_function" "%s" {
										function_name             = "%s"
										helm_chart                = "%s"
										helm_chart_service_name   = "%s"
										inference_port            = %d
										inference_url             = "%s"
										health                    = {
											uri                  = "%s"
											port                 = %d
											expected_status_code = 200
											timeout              = "PT10S"
											protocol             = "HTTP"
										}
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
										timeouts = {
											create = "1s"
										}
									}
									`,
					functionName,
					functionName,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmServicePort,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWriteUpdated),
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				ExpectError: regexp.MustCompile("timeout occurred"),
			},
			// Verify Function Force-Replace Update
			{
				Config: fmt.Sprintf(`
									resource "ngc_cloud_function" "%s" {
										function_name             = "%s"
										helm_chart                = "%s"
										helm_chart_service_name   = "%s"
										inference_port            = %d
										inference_url             = "%s"
										health                    = {
											uri                  = "%s"
											port                 = %d
											expected_status_code = 200
											timeout              = "PT10S"
											protocol             = "HTTP"
										}
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
									}
									`,
					functionName,
					functionName,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmServicePort,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWriteUpdated),
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "id"),
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "function_id"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "container_image"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "telemetries"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart", testutils.TestHelmUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name", testutils.TestHelmServiceName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestHelmInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestHelmAPIFormat),
					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration", testutils.TestHelmValueOverWriteUpdated),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestHelmHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "authorized_parties.#", "0"),
				),
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateFunctionStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"graceful_deletion", // Not assigned when import
				},
			},
		},
	})
}

func TestAccCloudFunctionResource_CreateHelmBasedFunctionVersionDeployWithClustersOptionSuccess(t *testing.T) {
	var functionName = testutils.TestCommonPrefix + "helm-based-function-version-with-clusters-option"
	var testCloudFunctionResourceFullPath = fmt.Sprintf("ngc_cloud_function.%s", functionName)

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
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
							api_body_format         = "%s"
							deployment_specifications = [
								{
									configuration           = "%s"
									instance_type           = "%s"
									clusters                = ["%s"]
									gpu_type                = "%s"
									max_instances           = 1
									min_instances           = 1
									max_request_concurrency = 1
								}
							]
						}
						`,
					functionName,
					functionName,
					functionInfo.Function.ID,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmServicePort,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWrite),
					testutils.TestInstanceType,
					testutils.TestClusters[0],
					testutils.TestGpuType,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify version ID exist
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					// Verify container attribute not exist
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "container_image"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "telemetries"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart", testutils.TestHelmUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name", testutils.TestHelmServiceName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestHelmInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestHelmAPIFormat),

					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.clusters.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.clusters.0", testutils.TestClusters[0]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration", testutils.TestHelmValueOverWrite),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestHelmHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "authorized_parties.#", "0"),
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
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
							api_body_format         = "%s"
							deployment_specifications = [
								{
									configuration           = "%s"
									clusters                = ["%s"]
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 2
									min_instances           = 1
									max_request_concurrency = 2
								}
							]
							authorized_parties = [
								{
									nca_id = "%s"
								},
								{
									nca_id = "%s"
								}
							]
						}
						`,
					functionName,
					functionName,
					functionInfo.Function.ID,
					testutils.TestHelmUri,
					testutils.TestHelmServiceName,
					testutils.TestHelmServicePort,
					testutils.TestHelmInferenceUrl,
					testutils.TestHelmHealthUri,
					testutils.TestHelmServicePort,
					testutils.TestHelmAPIFormat,
					testutils.EscapeJSON(t, testutils.TestHelmValueOverWrite),
					testutils.TestClusters[0],
					testutils.TestInstanceType,
					testutils.TestGpuType,
					testutils.TestAuthorizedParty1,
					testutils.TestAuthorizedParty2,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify version ID exist
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					// Verify container attribute not exist
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "container_image"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "telemetries"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart", testutils.TestHelmUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name", testutils.TestHelmServiceName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestHelmInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestHelmAPIFormat),
					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.clusters.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.clusters.0", testutils.TestClusters[0]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration", testutils.TestHelmValueOverWrite),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestHelmHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestHelmServicePort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "authorized_parties.#", "2"),
				),
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateFunctionStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"function_id",       // Not assigned when import
					"graceful_deletion", // Not assigned when import
				},
			},
		},
	})
}

func TestAccCloudFunctionResource_CreateContainerBasedFunctionVersionDeployWithClustersAndRegionOptionsSuccess(t *testing.T) {
	var functionName = testutils.TestCommonPrefix + "container-based-function-version-with-clusters-and-region-options"
	var testCloudFunctionResourceFullPath = fmt.Sprintf("ngc_cloud_function.%s", functionName)

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
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
							api_body_format         = "%s"
							deployment_specifications = [
								{
									clusters                = ["%s", "%s"]
									regions                 = ["%s", "%s"]
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 1
									min_instances           = 1
									max_request_concurrency = 1
								}
							]
						}
						`,
					functionName,
					functionName,
					functionInfo.Function.ID,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestClusters[0],
					testutils.TestClusters[1],
					testutils.TestRegions[0],
					testutils.TestRegions[1],
					testutils.TestInstanceType,
					testutils.TestGpuType,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "telemetries"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.clusters.#", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.clusters.0", testutils.TestClusters[0]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.clusters.1", testutils.TestClusters[1]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.regions.#", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.regions.0", testutils.TestRegions[0]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.regions.1", testutils.TestRegions[1]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "1"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestContainerHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "authorized_parties.#", "0"),
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
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
							api_body_format         = "%s"
							deployment_specifications = [
								{
									clusters                = ["%s", "%s"]
									regions                 = ["%s", "%s"]
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 2
									min_instances           = 1
									max_request_concurrency = 2
								}
							]
							authorized_parties = [
								{
									nca_id = "%s"
								},
								{
									nca_id = "%s"
								}
							]
						}
						`,
					functionName,
					functionName,
					functionInfo.Function.ID,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestClusters[0],
					testutils.TestClusters[1],
					testutils.TestRegions[0],
					testutils.TestRegions[1],
					testutils.TestInstanceType,
					testutils.TestGpuType,
					testutils.TestAuthorizedParty1,
					testutils.TestAuthorizedParty2,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "telemetries"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_id", functionInfo.Function.ID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.clusters.#", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.clusters.0", testutils.TestClusters[0]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.clusters.1", testutils.TestClusters[1]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.regions.#", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.regions.0", testutils.TestRegions[0]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.regions.1", testutils.TestRegions[1]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.instance_type", testutils.TestInstanceType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_instances", "2"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.min_instances", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.max_request_concurrency", "2"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.configuration"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestContainerHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "authorized_parties.#", "2"),
				),
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateFunctionStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"function_id",       // Not assigned when import,
					"graceful_deletion", // Not assigned when import
				},
			},
		},
	})
}

func TestAccCloudFunctionResource_CreateFunctionWithoutDeploymentSuccess(t *testing.T) {
	var functionName = testutils.TestCommonPrefix + "function-without-deployment"
	var testCloudFunctionResourceFullPath = fmt.Sprintf("ngc_cloud_function.%s", functionName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Verify Function Creation
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
							function_name           = "%s"
							container_image         = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
							api_body_format         = "%s"
							models                  = [
							    {
							    	name    = "%s"
									version = "%s"
									uri     = "%s"
								}
							]
						}
						`,
					functionName,
					functionName,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestModel1Name,
					testutils.TestModel1Version,
					testutils.TestModel1Uri,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "telemetries"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "0"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestContainerHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.name", testutils.TestModel1Name),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.version", testutils.TestModel1Version),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.uri", testutils.TestModel1FullyQualifiedUri),
				),
			},
			// Verify Function Update again won't change anything
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
							function_name           = "%s"
							container_image         = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
							api_body_format         = "%s"
							models                  = [
							    {
							    	name    = "%s"
									version = "%s"
									uri     = "%s"
								}
							]
						}
						`,
					functionName,
					functionName,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestModel1Name,
					testutils.TestModel1Version,
					testutils.TestModel1Uri,
				),
				ExpectNonEmptyPlan: false,
				PlanOnly:           true,
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateFunctionStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"function_id",       // Not assigned when import
					"graceful_deletion", // Not assigned when import
				},
			},
		},
	})
}

func TestAccCloudFunctionResource_CreateFunctionWithTelemetriesWithoutDeploymentSuccess(t *testing.T) {
	var functionName = testutils.TestCommonPrefix + "function-with-telemetries-without-deployment"
	var testCloudFunctionResourceFullPath = fmt.Sprintf("ngc_cloud_function.%s", functionName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Verify Function Creation
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
							function_name           = "%s"
							container_image         = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
							api_body_format         = "%s"
							models                  = [
							    {
							    	name    = "%s"
									version = "%s"
									uri     = "%s"
								}
							]
							telemetries = {
								logs_telemetry_id    = "%s"
								metrics_telemetry_id = "%s"
							}
						}
						`,
					functionName,
					functionName,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestModel1Name,
					testutils.TestModel1Version,
					testutils.TestModel1Uri,
					testutils.TestLogsTelemetryId,
					testutils.TestMetricsTelemetryId,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "0"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestContainerHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.name", testutils.TestModel1Name),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.version", testutils.TestModel1Version),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.uri", testutils.TestModel1FullyQualifiedUri),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "telemetries.logs_telemetry_id", testutils.TestLogsTelemetryId),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "telemetries.metrics_telemetry_id", testutils.TestMetricsTelemetryId),
				),
			},
			// Verify Function Update with removing telemetries
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
							function_name           = "%s"
							container_image         = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
							api_body_format         = "%s"
							models                  = [
							    {
							    	name    = "%s"
									version = "%s"
									uri     = "%s"
								}
							]
						}
						`,
					functionName,
					functionName,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestModel1Name,
					testutils.TestModel1Version,
					testutils.TestModel1Uri,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "telemetries"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "0"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestContainerHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.name", testutils.TestModel1Name),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.version", testutils.TestModel1Version),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.uri", testutils.TestModel1FullyQualifiedUri),
				),
			},
			// Verify Function Update again to bring back telemetries
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
							function_name           = "%s"
							container_image         = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
							api_body_format         = "%s"
							models                  = [
							    {
							    	name    = "%s"
									version = "%s"
									uri     = "%s"
								}
							]
							telemetries = {
								logs_telemetry_id    = "%s"
								metrics_telemetry_id = "%s"
							}
						}
						`,
					functionName,
					functionName,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestModel1Name,
					testutils.TestModel1Version,
					testutils.TestModel1Uri,
					testutils.TestLogsTelemetryId,
					testutils.TestMetricsTelemetryId,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "0"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestContainerHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.name", testutils.TestModel1Name),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.version", testutils.TestModel1Version),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.uri", testutils.TestModel1FullyQualifiedUri),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "telemetries.logs_telemetry_id", testutils.TestLogsTelemetryId),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "telemetries.metrics_telemetry_id", testutils.TestMetricsTelemetryId),
				),
			},
			// Verify Function Update again to remove telemetries
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
							function_name           = "%s"
							container_image         = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
							api_body_format         = "%s"
							models                  = [
							    {
							    	name    = "%s"
									version = "%s"
									uri     = "%s"
								}
							]
							telemetries = {
								logs_telemetry_id    = "%s"
								metrics_telemetry_id = "%s"
							}
						}
						`,
					functionName,
					functionName,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestModel1Name,
					testutils.TestModel1Version,
					testutils.TestModel1Uri,
					testutils.TestLogsTelemetryId,
					testutils.TestMetricsTelemetryId,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "0"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestContainerHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.name", testutils.TestModel1Name),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.version", testutils.TestModel1Version),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.uri", testutils.TestModel1FullyQualifiedUri),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "telemetries.logs_telemetry_id", testutils.TestLogsTelemetryId),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "telemetries.metrics_telemetry_id", testutils.TestMetricsTelemetryId),
				),
				ExpectNonEmptyPlan: false,
				PlanOnly:           true,
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateFunctionStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"function_id",       // Not assigned when import
					"graceful_deletion", // Not assigned when import
				},
			},
		},
	})
}

func TestAccCloudFunctionResource_CreateFunctionWithFullyQuailfiedArtifactsUrlFormatSuccess(t *testing.T) {
	var functionName = testutils.TestCommonPrefix + "function-with-fully-quailfied-artifacts-url-format"
	var testCloudFunctionResourceFullPath = fmt.Sprintf("ngc_cloud_function.%s", functionName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Verify Function Creation
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
							function_name           = "%s"
							container_image         = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
							api_body_format         = "%s"
							models                  = [
							    {
							    	name    = "%s"
									version = "%s"
									uri     = "%s"
								}
							]
						}
						`,
					functionName,
					functionName,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestModel1Name,
					testutils.TestModel1Version,
					testutils.TestModel1FullyQualifiedUri,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "telemetries"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "0"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestContainerHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.name", testutils.TestModel1Name),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.version", testutils.TestModel1Version),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.uri", testutils.TestModel1FullyQualifiedUri),
				),
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateFunctionStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"function_id",       // Not assigned when import
					"graceful_deletion", // Not assigned when import
				},
			},
		},
	})
}

func TestAccCloudFunctionResource_CreateFunctionWithLegacyArtifactUrlsFormatSuccess(t *testing.T) {
	var functionName = testutils.TestCommonPrefix + "function-with-legacy-artifact-urls-format"
	var testCloudFunctionResourceFullPath = fmt.Sprintf("ngc_cloud_function.%s", functionName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Verify Function Creation
			{
				Config: fmt.Sprintf(`
						resource "ngc_cloud_function" "%s" {
							function_name           = "%s"
							container_image         = "%s"
							inference_port          = %d
							inference_url           = "%s"
							health                    = {
								uri                  = "%s"
								port                 = %d
								expected_status_code = 200
								timeout              = "PT10S"
								protocol             = "HTTP"
							}
							api_body_format         = "%s"
							models                  = [
							    {
							    	name    = "%s"
									version = "%s"
									uri     = "%s"
								}
							]
						}
						`,
					functionName,
					functionName,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestModel1Name,
					testutils.TestModel1Version,
					testutils.TestModel1Uri,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testCloudFunctionResourceFullPath, "version_id"),

					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "helm_chart_service_name"),
					resource.TestCheckNoResourceAttr(testCloudFunctionResourceFullPath, "telemetries"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "nca_id", testutils.TestNcaID),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "function_name", functionName),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_image", testutils.TestContainerUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "0"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestContainerHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.name", testutils.TestModel1Name),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.version", testutils.TestModel1Version),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.uri", testutils.TestModel1FullyQualifiedUri),
				),
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateFunctionStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"function_id",       // Not assigned when import
					"graceful_deletion", // Not assigned when import
				},
			},
		},
	})
}
