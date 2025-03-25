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
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NvidiaCloudFunctionResourceContainerEnvironmentModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

type NvidiaCloudFunctionResourceHealthModel struct {
	Protocol           types.String `tfsdk:"protocol"`
	Uri                types.String `tfsdk:"uri"`
	Port               types.Int64  `tfsdk:"port"`
	Timeout            types.String `tfsdk:"timeout"`
	ExpectedStatusCode types.Int64  `tfsdk:"expected_status_code"`
}

func (m *NvidiaCloudFunctionResourceHealthModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"protocol":             types.StringType,
		"uri":                  types.StringType,
		"port":                 types.Int64Type,
		"timeout":              types.StringType,
		"expected_status_code": types.Int64Type,
	}
}

type NvidiaCloudFunctionResourceAuthorizedPartyModel struct {
	NcaID types.String `tfsdk:"nca_id"`
}

type NvidiaCloudFunctionResourceSecretModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type NvidiaCloudFunctionResourceResourceModel struct {
	Name    types.String `tfsdk:"name"`
	Uri     types.String `tfsdk:"uri"`
	Version types.String `tfsdk:"version"`
}

type NvidiaCloudFunctionResourceModelModel struct {
	Name    types.String `tfsdk:"name"`
	Uri     types.String `tfsdk:"uri"`
	Version types.String `tfsdk:"version"`
}

type NvidiaCloudFunctionResourceDeploymentSpecificationModel struct {
	GpuType               types.String `tfsdk:"gpu_type"`
	Backend               types.String `tfsdk:"backend"`
	MaxInstances          types.Int64  `tfsdk:"max_instances"`
	MinInstances          types.Int64  `tfsdk:"min_instances"`
	MaxRequestConcurrency types.Int64  `tfsdk:"max_request_concurrency"`
	Configuration         types.String `tfsdk:"configuration"`
	InstanceType          types.String `tfsdk:"instance_type"`
}

type NvidiaCloudFunctionResourceModel struct {
	Id                       types.String   `tfsdk:"id"`
	FunctionID               types.String   `tfsdk:"function_id"`
	VersionID                types.String   `tfsdk:"version_id"`
	NcaId                    types.String   `tfsdk:"nca_id"`
	FunctionName             types.String   `tfsdk:"function_name"`
	InferencePort            types.Int64    `tfsdk:"inference_port"`
	HelmChart                types.String   `tfsdk:"helm_chart"`
	HelmChartServiceName     types.String   `tfsdk:"helm_chart_service_name"`
	ContainerImage           types.String   `tfsdk:"container_image"`
	ContainerArgs            types.String   `tfsdk:"container_args"`
	ContainerEnvironment     types.Set      `tfsdk:"container_environment"`
	InferenceUrl             types.String   `tfsdk:"inference_url"`
	HealthUri                types.String   `tfsdk:"health_uri"` // Deprecated
	Health                   types.Object   `tfsdk:"health"`
	APIBodyFormat            types.String   `tfsdk:"api_body_format"`
	DeploymentSpecifications types.List     `tfsdk:"deployment_specifications"`
	Tags                     types.Set      `tfsdk:"tags"`
	Description              types.String   `tfsdk:"description"`
	Models                   types.Set      `tfsdk:"models"`
	Resources                types.Set      `tfsdk:"resources"`
	FunctionType             types.String   `tfsdk:"function_type"`
	KeepFailedResource       types.Bool     `tfsdk:"keep_failed_resource"`
	Timeouts                 timeouts.Value `tfsdk:"timeouts"`
	Secrets                  types.Set      `tfsdk:"secrets"`
	AuthorizedParties        types.Set      `tfsdk:"authorized_parties"`
}
