//  SPDX-FileCopyrightText: Copyright (c) 2024 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
//  SPDX-License-Identifier: LicenseRef-NvidiaProprietary

//  NVIDIA CORPORATION, its affiliates and licensors retain all intellectual
//  property and proprietary rights in and to this material, related
//  documentation and any modifications thereto. Any use, reproduction,
//  disclosure or distribution of this material and related documentation
//  without an express license agreement from NVIDIA CORPORATION or
//  its affiliates is strictly prohibited.

package provider

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab-master.nvidia.com/nvb/core/terraform-provider-ngc/internal/provider/testutils"
)

const (
	TELEMETRY_ENDPOINT = "https://otlp-gateway-prod-us-west-0.grafana.net/otlp"
	TELEMETRY_PROTOCOL = "HTTP"
	TELEMETRY_PROVIDER = "GRAFANA_CLOUD"
)

var TELEMETRY_TYPES = []string{"METRICS", "LOGS"}

func generateTelemetryResourceConfig(telemetryName string, telemetryTypes []string) string {
	telemetryTypesJson, err := json.Marshal(telemetryTypes)
	if err != nil {
		panic("Error marshalling in telemetryTypes: " + strings.Join(telemetryTypes, ",") + " " + err.Error())
	}

	return fmt.Sprintf(`
		resource "ngc_cloud_function_telemetry" "%s" {
			endpoint           = "%s"
			protocol           = "%s"
			telemetry_provider = "%s"
			types              = %s
			secret = {
				name  = "%s"
				value = "123"
			}
		}
	`, telemetryName, TELEMETRY_ENDPOINT, TELEMETRY_PROTOCOL, TELEMETRY_PROVIDER, string(telemetryTypesJson), telemetryName)
}

func generateFunctionTelemetryStateResourceId(resourceName string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		var rawState map[string]string
		for _, m := range state.Modules {
			if len(m.Resources) > 0 {
				if v, ok := m.Resources[resourceName]; ok {
					rawState = v.Primary.Attributes
				}
			}
		}
		return fmt.Sprintf("%s", rawState["id"]), nil
	}
}

func TestAccCloudFunctionTelemetryResource_CreateAndUpdateAndDeleteTelemetrySuccess(t *testing.T) {
	var telemetryName = testutils.TestCommonPrefix + "telemetry-resource"
	var testCloudFunctionTelemetryResourceFullPath = fmt.Sprintf("ngc_cloud_function_telemetry.%s", telemetryName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: generateTelemetryResourceConfig(telemetryName, TELEMETRY_TYPES),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "name", telemetryName),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "endpoint", TELEMETRY_ENDPOINT),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "protocol", TELEMETRY_PROTOCOL),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "telemetry_provider", TELEMETRY_PROVIDER),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "types.#", "2"),
					resource.TestCheckTypeSetElemAttr(testCloudFunctionTelemetryResourceFullPath, "types.*", TELEMETRY_TYPES[0]),
					resource.TestCheckTypeSetElemAttr(testCloudFunctionTelemetryResourceFullPath, "types.*", TELEMETRY_TYPES[1]),
				),
			},
			// Verify Telemetry Update again won't change anything
			{
				Config: generateTelemetryResourceConfig(telemetryName, TELEMETRY_TYPES),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "name", telemetryName),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "endpoint", TELEMETRY_ENDPOINT),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "protocol", TELEMETRY_PROTOCOL),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "telemetry_provider", TELEMETRY_PROVIDER),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "types.#", "2"),
					resource.TestCheckTypeSetElemAttr(testCloudFunctionTelemetryResourceFullPath, "types.*", TELEMETRY_TYPES[0]),
					resource.TestCheckTypeSetElemAttr(testCloudFunctionTelemetryResourceFullPath, "types.*", TELEMETRY_TYPES[1]),
				),
			},
			// Verify Telemetry Update
			{
				Config: generateTelemetryResourceConfig(telemetryName, []string{"LOGS"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "name", telemetryName),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "endpoint", TELEMETRY_ENDPOINT),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "protocol", TELEMETRY_PROTOCOL),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "telemetry_provider", TELEMETRY_PROVIDER),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "types.#", "1"),
					resource.TestCheckTypeSetElemAttr(testCloudFunctionTelemetryResourceFullPath, "types.*", "LOGS"),
				),
			},
			// Verify Telemetry Import
			{
				ResourceName:      testCloudFunctionTelemetryResourceFullPath,
				ImportStateIdFunc: generateFunctionTelemetryStateResourceId(testCloudFunctionTelemetryResourceFullPath),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"secret",
				},
			},
		},
	})
}

func TestAccCloudFunctionTelemetryResource_Fail(t *testing.T) {
	var telemetryName = testutils.TestCommonPrefix + "telemetry-resource-fail"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
						resource "ngc_cloud_function_telemetry" "%s" {
							protocol           = "%s"
							telemetry_provider = "%s"
							types              = ["%s"]
							secret = {
								name  = "%s"
								value = "123"
							}
						}
					`, telemetryName, TELEMETRY_PROTOCOL, TELEMETRY_PROVIDER, "LOGS", telemetryName,
				),
				ExpectError: regexp.MustCompile("The argument \"endpoint\" is required, but no definition was found."),
			},
			{
				Config: fmt.Sprintf(
					`
							resource "ngc_cloud_function_telemetry" "%s" {
								endpoint           = "%s"
								telemetry_provider = "%s"
								types              = ["%s"]
								secret = {
									name  = "%s"
									value = "123"
								}
							}
						`, telemetryName, TELEMETRY_ENDPOINT, TELEMETRY_PROVIDER, "LOGS", telemetryName,
				),
				ExpectError: regexp.MustCompile("The argument \"protocol\" is required, but no definition was found."),
			},
			{
				Config: fmt.Sprintf(
					`
							resource "ngc_cloud_function_telemetry" "%s" {
								endpoint           = "%s"
								protocol           = "%s"
								types              = ["%s"]
								secret = {
									name  = "%s"
									value = "123"
								}
							}
						`, telemetryName, TELEMETRY_ENDPOINT, TELEMETRY_PROTOCOL, "LOGS", telemetryName,
				),
				ExpectError: regexp.MustCompile("The argument \"telemetry_provider\" is required, but no definition was found."),
			},
			{
				Config: fmt.Sprintf(
					`
							resource "ngc_cloud_function_telemetry" "%s" {
								endpoint           = "%s"
								protocol           = "%s"
								telemetry_provider = "%s"
								types              = ["%s"]
								secret = {
									value = "123"
								}
							}
						`, telemetryName, TELEMETRY_ENDPOINT, TELEMETRY_PROTOCOL, TELEMETRY_PROVIDER, "LOGS",
				),
				ExpectError: regexp.MustCompile("Inappropriate value for attribute \"secret\": attribute \"name\" is required."),
			},
			{
				Config: fmt.Sprintf(
					`
							resource "ngc_cloud_function_telemetry" "%s" {
								endpoint           = "%s"
								protocol           = "%s"
								telemetry_provider = "%s"
								types              = ["%s"]
								secret = {
									name  = "%s"
								}
							}
						`, telemetryName, TELEMETRY_ENDPOINT, TELEMETRY_PROTOCOL, TELEMETRY_PROVIDER, "LOGS", telemetryName,
				),
				ExpectError: regexp.MustCompile("Inappropriate value for attribute \"secret\": attribute \"value\" is required."),
			},
		},
	})
}
