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
					testCloudFunctionResourceName,
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
					testCloudFunctionResourceName,
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
						}
						`,
					testCloudFunctionResourceName,
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
					testCloudFunctionResourceName,
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
					testCloudFunctionResourceName,
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
					testCloudFunctionResourceName,
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
				ResourceName:            testCloudFunctionResourceFullPath,
				ImportStateIdFunc:       generateStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
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
					testutils.TestHelmServicePort,
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
									backend                 = "%s"
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
					testCloudFunctionResourceName,
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
					testutils.TestBackend,
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

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "authorized_parties.#", "2"),
				),
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"function_id", // Not assigned when import
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
							secrets = [
								{
									name  = "%s"
									value = "test-raw"
								},
								{
									name  = "%s"
									value = <<EOF
									{
										"AWS_REGION": "us-west-2",
										"AWS_BUCKET": "content",
										"AWS_ACCESS_KEY_ID": "content-key-id",
										"AWS_SECRET_ACCESS_KEY": "content-access-key",
										"AWS_SESSION_TOKEN": "content-session-token"
									}
									EOF
								},
								{
									name  = "%s"
									value = <<EOF
									{
										"AWS_ACCESS_KEY_ID" : "s3.us-west-2-key-id",
										"AWS_SECRET_ACCESS_KEY" : "s3.us-west-2-access-key"
									}
									EOF
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
					testCloudFunctionResourceName,
					functionName,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
					testutils.TestTags[0],
					testutils.TestTags[1],
					testutils.TestContainerEnvironmentVariables[0].Key,
					testutils.TestContainerEnvironmentVariables[0].Value,
					testutils.TestSecretNames[0],
					testutils.TestSecretNames[1],
					testutils.TestSecretNames[2],
					testutils.TestAuthorizedParty1,
					testutils.TestAuthorizedParty2,
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
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
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

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "secrets.0.name", testutils.TestSecretNames[0]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "secrets.1.name", testutils.TestSecretNames[1]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "secrets.2.name", testutils.TestSecretNames[2]),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_environment.0.key", testutils.TestContainerEnvironmentVariables[0].Key),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_environment.0.value", testutils.TestContainerEnvironmentVariables[0].Value),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.protocol", "HTTP"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.uri", testutils.TestContainerHealthUri),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.port", strconv.Itoa(testutils.TestContainerPort)),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.timeout", "PT10S"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "health.expected_status_code", "200"),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "authorized_parties.#", "2"),
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
									backend                 = "%s"
									instance_type           = "%s"
									gpu_type                = "%s"
									max_instances           = 2
									min_instances           = 1
									max_request_concurrency = 2
								}
							]
							tags = ["%s","%s"]
							container_environment = [
								{
									key   = "%s"
									value = "%s"
								}
							]
							secrets = [
								{
									name  = "%s"
									value = "test-raw"
								},
								{
									name  = "%s"
									value = <<EOF
									{
										"AWS_REGION": "us-west-2",
										"AWS_BUCKET": "content",
										"AWS_ACCESS_KEY_ID": "content-key-id",
										"AWS_SECRET_ACCESS_KEY": "content-access-key",
										"AWS_SESSION_TOKEN": "content-session-token"
									}
									EOF
								},
								{
									name  = "%s"
									value = <<EOF
									{
										"AWS_ACCESS_KEY_ID" : "s3.us-west-2-key-id",
										"AWS_SECRET_ACCESS_KEY" : "s3.us-west-2-access-key"
									}
									EOF
								}
							]
						}
						`,
					testCloudFunctionResourceName,
					functionName,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
					testutils.TestTags[0],
					testutils.TestTags[1],
					testutils.TestContainerEnvironmentVariables[0].Key,
					testutils.TestContainerEnvironmentVariables[0].Value,
					testutils.TestSecretNames[0],
					testutils.TestSecretNames[1],
					testutils.TestSecretNames[2],
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
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "api_body_format", testutils.TestContainerAPIFormat),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "tags.0", testutils.TestTags[0]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "tags.1", testutils.TestTags[1]),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "secrets.0.name", testutils.TestSecretNames[0]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "secrets.1.name", testutils.TestSecretNames[1]),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "secrets.2.name", testutils.TestSecretNames[2]),

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_environment.0.key", testutils.TestContainerEnvironmentVariables[0].Key),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "container_environment.0.value", testutils.TestContainerEnvironmentVariables[0].Value),

					// Verify number of deployment_specifications
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.#", "1"),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.gpu_type", testutils.TestGpuType),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "deployment_specifications.0.backend", testutils.TestBackend),
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

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "authorized_parties.#", "0"),
				),
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"secrets", // Won't retrieve from API.
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
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
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
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
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
									backend                 = "%s"
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
					testCloudFunctionResourceName,
					functionName,
					functionInfo.Function.ID,
					testutils.TestContainerUri,
					testutils.TestContainerPort,
					testutils.TestContainerInferenceUrl,
					testutils.TestContainerHealthUri,
					testutils.TestContainerPort,
					testutils.TestContainerAPIFormat,
					testutils.TestBackend,
					testutils.TestInstanceType,
					testutils.TestGpuType,
					testutils.TestAuthorizedParty1,
					testutils.TestAuthorizedParty2,
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
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "inference_url", testutils.TestContainerInferenceUrl),
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
				ImportStateIdFunc: generateStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"function_id", // Not assigned when import
				},
			},
		},
	})
}

func TestAccCloudFunctionResource_FunctionWithoutDeployment(t *testing.T) {
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
					testCloudFunctionResourceName,
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
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.0.uri", testutils.TestModel1Uri),
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
					testCloudFunctionResourceName,
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

					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.1.name", testutils.TestModel1Name),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.1.version", testutils.TestModel1Version),
					resource.TestCheckResourceAttr(testCloudFunctionResourceFullPath, "models.1.uri", testutils.TestModel1Uri),
				),
				ExpectNonEmptyPlan: false,
				PlanOnly:           true,
			},
			// Verify Function Import
			{
				ResourceName:      testCloudFunctionResourceFullPath,
				ImportStateIdFunc: generateStateResourceId(testCloudFunctionResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"function_id", // Not assigned when import
				},
			},
		},
	})
}
