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
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testCloudFunctionTelemetryDatasourceName = "terraform-cloud-function-telemetry-datasource"
var testCloudFunctionTelemetryDatasourceFullPath = fmt.Sprintf("data.ngc_cloud_function_telemetry.%s", testCloudFunctionTelemetryDatasourceName)

func TestAccCloudFunctionTelemetryDataSource_Success(t *testing.T) {
	var telemetryResourceName = "terraform-cloud-function-telemetry-resource"
	var testCloudFunctionTelemetryResourceFullPath = fmt.Sprintf("ngc_cloud_function_telemetry.%s", telemetryResourceName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "ngc_cloud_function_telemetry" "%s" {
						endpoint           = "%s"
						protocol           = "%s"
						telemetry_provider = "%s"
						types              = ["%s", "%s"]
						secret = {
							name  = "%s"
							value = "123"
						}
					}

					data "ngc_cloud_function_telemetry" "%s" {
						id = ngc_cloud_function_telemetry.%s.id
					}
				`, telemetryResourceName, TELEMETRY_ENDPOINT, TELEMETRY_PROTOCOL, TELEMETRY_PROVIDER, TELEMETRY_TYPES[0], TELEMETRY_TYPES[1], telemetryResourceName, testCloudFunctionTelemetryDatasourceName, telemetryResourceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check resource attributes
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "name", telemetryResourceName),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "endpoint", TELEMETRY_ENDPOINT),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "protocol", TELEMETRY_PROTOCOL),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "telemetry_provider", TELEMETRY_PROVIDER),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryResourceFullPath, "types.#", "2"),
					resource.TestCheckTypeSetElemAttr(testCloudFunctionTelemetryResourceFullPath, "types.*", TELEMETRY_TYPES[0]),
					resource.TestCheckTypeSetElemAttr(testCloudFunctionTelemetryResourceFullPath, "types.*", TELEMETRY_TYPES[1]),

					// Check datasource attributes
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryDatasourceFullPath, "name", telemetryResourceName),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryDatasourceFullPath, "endpoint", TELEMETRY_ENDPOINT),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryDatasourceFullPath, "protocol", TELEMETRY_PROTOCOL),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryDatasourceFullPath, "telemetry_provider", TELEMETRY_PROVIDER),
					resource.TestCheckResourceAttr(testCloudFunctionTelemetryDatasourceFullPath, "types.#", "2"),
					resource.TestCheckTypeSetElemAttr(testCloudFunctionTelemetryDatasourceFullPath, "types.*", TELEMETRY_TYPES[0]),
					resource.TestCheckTypeSetElemAttr(testCloudFunctionTelemetryDatasourceFullPath, "types.*", TELEMETRY_TYPES[1]),

					// Check that IDs match
					resource.TestCheckResourceAttrPair(testCloudFunctionTelemetryResourceFullPath, "id", testCloudFunctionTelemetryDatasourceFullPath, "id"),
					resource.TestCheckResourceAttrPair(testCloudFunctionTelemetryResourceFullPath, "created_at", testCloudFunctionTelemetryDatasourceFullPath, "created_at"),
				),
			},
		},
	})
}
