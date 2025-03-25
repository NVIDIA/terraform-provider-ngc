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
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab-master.nvidia.com/nvb/core/terraform-provider-ngc/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &NvidiaCloudFunctionDataSource{}

func NewNvidiaCloudFunctionDataSource() datasource.DataSource {
	return &NvidiaCloudFunctionDataSource{}
}

// NvidiaCloudFunctionDataSource defines the data source implementation.
type NvidiaCloudFunctionDataSource struct {
	client *utils.NVCFClient
}

// NvidiaCloudFunctionDataSourceModel describes the data source data model.
type NvidiaCloudFunctionDataSourceModel struct {
	FunctionID               types.String                            `tfsdk:"function_id"`
	VersionID                types.String                            `tfsdk:"version_id"`
	NcaId                    types.String                            `tfsdk:"nca_id"`
	FunctionName             types.String                            `tfsdk:"function_name"`
	HelmChart                types.String                            `tfsdk:"helm_chart"`
	HelmChartServiceName     types.String                            `tfsdk:"helm_chart_service_name"`
	InferencePort            types.Int64                             `tfsdk:"inference_port"`
	ContainerImage           types.String                            `tfsdk:"container_image"`
	ContainerArgs            types.String                            `tfsdk:"container_args"`
	ContainerEnvironment     types.Set                               `tfsdk:"container_environment"`
	InferenceUrl             types.String                            `tfsdk:"inference_url"`
	HealthUri                types.String                            `tfsdk:"health_uri"`
	Health                   *NvidiaCloudFunctionResourceHealthModel `tfsdk:"health"`
	APIBodyFormat            types.String                            `tfsdk:"api_body_format"`
	DeploymentSpecifications types.List                              `tfsdk:"deployment_specifications"`
	Tags                     types.Set                               `tfsdk:"tags"`
	Description              types.String                            `tfsdk:"description"`
	Models                   types.Set                               `tfsdk:"models"`
	Resources                types.Set                               `tfsdk:"resources"`
	FunctionType             types.String                            `tfsdk:"function_type"`
	AuthorizedParties        types.Set                               `tfsdk:"authorized_parties"`
}

func (d *NvidiaCloudFunctionDataSource) updateNvidiaCloudFunctionDataSourceModel(
	ctx context.Context, diag *diag.Diagnostics,
	data *NvidiaCloudFunctionDataSourceModel,
	functionInfo *utils.NvidiaCloudFunctionInfo,
	functionDeployment *utils.NvidiaCloudFunctionDeployment,
	functionAuthorizedParties []utils.AuthorizedParty,
) {
	data.VersionID = types.StringValue(functionInfo.VersionID)
	data.FunctionName = types.StringValue(functionInfo.Name)
	data.FunctionID = types.StringValue(functionInfo.ID)
	data.InferencePort = types.Int64Value(int64(functionInfo.InferencePort))

	if functionInfo.APIBodyFormat != "" {
		data.APIBodyFormat = types.StringValue(functionInfo.APIBodyFormat)
	}

	if functionInfo.InferenceURL != "" {
		data.InferenceUrl = types.StringValue(functionInfo.InferenceURL)
	}

	if functionInfo.NcaID != "" {
		data.NcaId = types.StringValue(functionInfo.NcaID)
	}

	if functionInfo.Name != "" {
		data.FunctionName = types.StringValue(functionInfo.Name)
	}

	if functionInfo.HealthURI != "" {
		data.HealthUri = types.StringValue(functionInfo.HealthURI)
	}

	if functionInfo.HelmChartServiceName != "" {
		data.HelmChartServiceName = types.StringValue(functionInfo.HelmChartServiceName)
	}

	if functionInfo.HelmChart != "" {
		data.HelmChart = types.StringValue(functionInfo.HelmChart)
	}

	if functionInfo.ContainerImage != "" {
		data.ContainerImage = types.StringValue(functionInfo.ContainerImage)
	}

	if functionInfo.ContainerArgs != "" {
		data.ContainerArgs = types.StringValue(functionInfo.ContainerArgs)
	}

	if functionInfo.FunctionType != "" {
		data.FunctionType = types.StringValue(functionInfo.FunctionType)
	}

	if functionInfo.Description != "" {
		data.Description = types.StringValue(functionInfo.Description)
	}

	if functionDeployment.DeploymentSpecifications != nil {
		deploymentSpecifications := make([]NvidiaCloudFunctionResourceDeploymentSpecificationModel, 0)

		for _, v := range functionDeployment.DeploymentSpecifications {
			deploymentSpecification := NvidiaCloudFunctionResourceDeploymentSpecificationModel{
				Backend:               types.StringValue(v.Backend),
				InstanceType:          types.StringValue(v.InstanceType),
				GpuType:               types.StringValue(v.Gpu),
				MaxInstances:          types.Int64Value(int64(v.MaxInstances)),
				MinInstances:          types.Int64Value(int64(v.MinInstances)),
				MaxRequestConcurrency: types.Int64Value(int64(v.MaxRequestConcurrency)),
			}

			if v.Configuration != nil {
				configuration, _ := json.Marshal(v.Configuration)
				deploymentSpecification.Configuration = types.StringValue(string(configuration))
			}

			deploymentSpecifications = append(deploymentSpecifications, deploymentSpecification)
		}
		deploymentSpecificationsListType, deploymentSpecificationsListTypeDiag := types.ListValueFrom(ctx, deploymentSpecificationsSchema().NestedObject.Type(), deploymentSpecifications)
		diag.Append(deploymentSpecificationsListTypeDiag...)
		data.DeploymentSpecifications = deploymentSpecificationsListType
	}

	tags, tagsSetFromDiag := types.SetValueFrom(ctx, types.StringType, functionInfo.Tags)
	diag.Append(tagsSetFromDiag...)
	data.Tags = tags

	data.Health = &NvidiaCloudFunctionResourceHealthModel{
		Protocol:           types.StringValue(functionInfo.Health.Protocol),
		Uri:                types.StringValue(functionInfo.Health.URI),
		Port:               types.Int64Value(int64(functionInfo.Health.Port)),
		Timeout:            types.StringValue(functionInfo.Health.Timeout),
		ExpectedStatusCode: types.Int64Value(int64(functionInfo.Health.ExpectedStatusCode)),
	}

	if functionInfo.ContainerEnvironment != nil {
		containerEnvironments := make([]NvidiaCloudFunctionResourceContainerEnvironmentModel, 0)
		for _, v := range functionInfo.ContainerEnvironment {
			containerEnvironment := NvidiaCloudFunctionResourceContainerEnvironmentModel{
				Key:   types.StringValue(v.Key),
				Value: types.StringValue(v.Value),
			}

			containerEnvironments = append(containerEnvironments, containerEnvironment)
		}
		containerEnvironmentsSetType, containerEnvironmentsSetTypeDiag := types.SetValueFrom(ctx, containerEnvironmentsSchema().NestedObject.Type(), containerEnvironments)
		diag.Append(containerEnvironmentsSetTypeDiag...)
		data.ContainerEnvironment = containerEnvironmentsSetType
	}

	if functionInfo.Resources != nil {
		resources := make([]NvidiaCloudFunctionResourceResourceModel, 0)
		for _, v := range functionInfo.Resources {
			resource := NvidiaCloudFunctionResourceResourceModel{
				Name:    types.StringValue(v.Name),
				Uri:     types.StringValue(v.URI),
				Version: types.StringValue(v.Version),
			}
			resources = append(resources, resource)
		}
		resourcesSetType, resourcesSetTypeDiag := types.SetValueFrom(ctx, resourcesSchema().NestedObject.Type(), resources)
		diag.Append(resourcesSetTypeDiag...)
		data.Resources = resourcesSetType
	}

	if functionInfo.Models != nil {
		models := make([]NvidiaCloudFunctionResourceModelModel, 0)
		for _, v := range functionInfo.Models {
			model := NvidiaCloudFunctionResourceModelModel{
				Name:    types.StringValue(v.Name),
				Uri:     types.StringValue(v.URI),
				Version: types.StringValue(v.Version),
			}
			models = append(models, model)
		}
		modelsSetType, modelsSetTypeDiag := types.SetValueFrom(ctx, modelsSchema().NestedObject.Type(), models)
		diag.Append(modelsSetTypeDiag...)
		data.Models = modelsSetType
	}

	if len(functionAuthorizedParties) > 0 {
		parties := make([]NvidiaCloudFunctionResourceAuthorizedPartyModel, 0)
		for _, v := range functionAuthorizedParties {
			party := NvidiaCloudFunctionResourceAuthorizedPartyModel{
				NcaID: types.StringValue(v.NcaID),
			}
			parties = append(parties, party)
		}
		partiesSetType, partiesSetTypeDiag := types.SetValueFrom(ctx, authorizedPartiesSchema().NestedObject.Type(), parties)
		diag.Append(partiesSetTypeDiag...)
		data.AuthorizedParties = partiesSetType
	}
}

func (d *NvidiaCloudFunctionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_function"
}

func (d *NvidiaCloudFunctionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Nvidia Cloud Function Data Source",

		Attributes: map[string]schema.Attribute{
			"function_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Function ID",
			},
			"version_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Function Version ID",
			},
			"nca_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "NCA ID",
			},
			"function_name": schema.StringAttribute{
				MarkdownDescription: "Function name",
				Optional:            true,
			},
			"helm_chart": schema.StringAttribute{
				MarkdownDescription: "Helm chart registry uri",
				Optional:            true,
			},
			"helm_chart_service_name": schema.StringAttribute{
				MarkdownDescription: "Target service name",
				Optional:            true,
			},
			"inference_port": schema.Int64Attribute{
				MarkdownDescription: "Target port, will be service port or container port base on function-based",
				Optional:            true,
			},
			"container_image": schema.StringAttribute{
				MarkdownDescription: "Container image uri",
				Optional:            true,
			},
			"container_environment": containerEnvironmentsSchema(),
			"container_args": schema.StringAttribute{
				MarkdownDescription: "Args to be passed when launching the container",
				Optional:            true,
			},
			"inference_url": schema.StringAttribute{
				MarkdownDescription: "Service endpoint Path.",
				Optional:            true,
			},
			"health_uri": schema.StringAttribute{
				MarkdownDescription: "Service health endpoint Path. Default is \"/v2/health/ready\"",
				Optional:            true,
				Computed:            true,
				DeprecationMessage:  "The parameter is deprecated. Please replace it with `health`",
			},
			"health":             healthSchema(),
			"models":             modelsSchema(),
			"resources":          resourcesSchema(),
			"authorized_parties": authorizedPartiesSchema(),
			"tags": schema.SetAttribute{
				MarkdownDescription: "Tags of the function.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the function",
				Optional:            true,
				Computed:            true,
			},
			"function_type": schema.StringAttribute{
				MarkdownDescription: "Optional function type, used to indicate a STREAMING function. Defaults is \"DEFAULT\".",
				Optional:            true,
				Computed:            true,
			},
			"api_body_format": schema.StringAttribute{
				MarkdownDescription: "API Body Format. Default is \"CUSTOM\"",
				Optional:            true,
				Computed:            true,
			},
			"deployment_specifications": deploymentSpecificationsSchema(),
		},
	}
}

func (d *NvidiaCloudFunctionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	ngcClient, ok := req.ProviderData.(*utils.NGCClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *NGCClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = ngcClient.NVCFClient()
}

func (d *NvidiaCloudFunctionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NvidiaCloudFunctionDataSourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var listNvidiaCloudFunctionVersionsResponse, err = d.client.ListNvidiaCloudFunctionVersions(ctx, data.FunctionID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read Cloud Function versions",
			"Got unexpected result when reading Cloud Function",
		)
		return
	}

	versionNotFound := true
	var functionVersion utils.NvidiaCloudFunctionInfo

	for _, f := range listNvidiaCloudFunctionVersionsResponse.Functions {
		if f.ID == data.FunctionID.ValueString() && f.VersionID == data.VersionID.ValueString() {
			functionVersion = f
			versionNotFound = false
			break
		}
	}

	if versionNotFound {
		resp.Diagnostics.AddError("Version ID Not Found Error", fmt.Sprintf("Unable to find the target version ID %s", data.VersionID.ValueString()))
		return
	}

	readNvidiaCloudFunctionDeploymentResponse, err := d.client.ReadNvidiaCloudFunctionDeployment(ctx, data.FunctionID.ValueString(), data.VersionID.ValueString())

	if err != nil {
		// FIXME: extract error messsage to constants.
		if err.Error() != "failed to find function deployment" {
			resp.Diagnostics.AddError(
				"Failed to read Cloud Function deployment",
				err.Error(),
			)
			return
		}
	}

	getFunctionAuthorizationResponse, err := d.client.GetFunctionAuthorization(ctx, data.FunctionID.ValueString(), data.VersionID.ValueString())

	if err != nil {
		// FIXME: extract error messsage to constants.
		resp.Diagnostics.AddError(
			"Failed to read Cloud Function authorized parties",
			err.Error(),
		)
		return
	}

	d.updateNvidiaCloudFunctionDataSourceModel(ctx, &resp.Diagnostics, &data, &functionVersion, &readNvidiaCloudFunctionDeploymentResponse.Deployment, getFunctionAuthorizationResponse.Function.AuthorizedParties)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
