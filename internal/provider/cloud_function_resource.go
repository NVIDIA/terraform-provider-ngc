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
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gitlab-master.nvidia.com/nvb/core/terraform-provider-ngc/internal/provider/utils"
)

const DEFAULT_TIMEOUT_SEC = 60 * 60

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &NvidiaCloudFunctionResource{}
var _ resource.ResourceWithImportState = &NvidiaCloudFunctionResource{}

func NewNvidiaCloudFunctionResource() resource.Resource {
	return &NvidiaCloudFunctionResource{}
}

// NvidiaCloudFunctionResource defines the resource implementation.
type NvidiaCloudFunctionResource struct {
	client *utils.NVCFClient
}

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
	ContainerEnvironment     types.List     `tfsdk:"container_environment"`
	InferenceUrl             types.String   `tfsdk:"inference_url"`
	HealthUri                types.String   `tfsdk:"health_uri"` // Deprecated
	Health                   types.Object   `tfsdk:"health"`
	APIBodyFormat            types.String   `tfsdk:"api_body_format"`
	DeploymentSpecifications types.List     `tfsdk:"deployment_specifications"`
	Tags                     types.Set      `tfsdk:"tags"`
	Description              types.String   `tfsdk:"description"`
	Models                   types.List     `tfsdk:"models"`
	Resources                types.List     `tfsdk:"resources"`
	FunctionType             types.String   `tfsdk:"function_type"`
	KeepFailedResource       types.Bool     `tfsdk:"keep_failed_resource"`
	Timeouts                 timeouts.Value `tfsdk:"timeouts"`
}

func (r *NvidiaCloudFunctionResource) updateNvidiaCloudFunctionResourceModel(
	ctx context.Context, diag *diag.Diagnostics,
	userProvideFunctionID types.String,
	data *NvidiaCloudFunctionResourceModel,
	functionInfo *utils.NvidiaCloudFunctionInfo,
	functionDeployment *utils.NvidiaCloudFunctionDeployment) {
	data.Id = types.StringValue(functionInfo.ID)
	data.VersionID = types.StringValue(functionInfo.VersionID)
	data.FunctionID = userProvideFunctionID

	data.InferencePort = types.Int64Value(int64(functionInfo.InferencePort))

	if data.KeepFailedResource.IsNull() || data.KeepFailedResource.IsUnknown() {
		data.KeepFailedResource = types.BoolValue(false)
	}

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

	deploymentSpecifications := make([]NvidiaCloudFunctionResourceDeploymentSpecificationModel, 0)

	if functionDeployment != nil {
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
	}

	deploymentSpecificationsSetType, deploymentSpecificationsSetTypeDiag := types.ListValueFrom(ctx, deploymentSpecificationsSchema().NestedObject.Type(), deploymentSpecifications)
	diag.Append(deploymentSpecificationsSetTypeDiag...)
	data.DeploymentSpecifications = deploymentSpecificationsSetType

	tags, tagsSetFromDiag := types.SetValueFrom(ctx, types.StringType, functionInfo.Tags)
	diag.Append(tagsSetFromDiag...)
	data.Tags = tags

	if functionInfo.Health != nil {
		healthObject := &NvidiaCloudFunctionResourceHealthModel{
			Protocol:           types.StringValue(functionInfo.Health.Protocol),
			Uri:                types.StringValue(functionInfo.Health.URI),
			Port:               types.Int64Value(int64(functionInfo.Health.Port)),
			Timeout:            types.StringValue(functionInfo.Health.Timeout),
			ExpectedStatusCode: types.Int64Value(int64(functionInfo.Health.ExpectedStatusCode)),
		}

		healthObjectType, healthObjectTypeDiag := types.ObjectValueFrom(ctx, healthObject.attrTypes(), healthObject)
		diag.Append(healthObjectTypeDiag...)
		data.Health = healthObjectType
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
		containerEnvironmentsSetType, containerEnvironmentsSetTypeDiag := types.ListValueFrom(ctx, containerEnvironmentsSchema().NestedObject.Type(), containerEnvironments)
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
		resourcesSetType, resourcesSetTypeDiag := types.ListValueFrom(ctx, resourcesSchema().NestedObject.Type(), resources)
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
		modelsSetType, modelsSetTypeDiag := types.ListValueFrom(ctx, resourcesSchema().NestedObject.Type(), models)
		diag.Append(modelsSetTypeDiag...)
		data.Models = modelsSetType
	}
}

func createDeployment(ctx context.Context, data NvidiaCloudFunctionResourceModel, diag *diag.Diagnostics, client utils.NVCFClient, function utils.NvidiaCloudFunctionInfo) (utils.NvidiaCloudFunctionDeployment, bool) {
	var functionDeployment utils.NvidiaCloudFunctionDeployment

	if !data.DeploymentSpecifications.IsNull() && len(data.DeploymentSpecifications.Elements()) > 0 {
		deploymentSpecifications := make([]NvidiaCloudFunctionResourceDeploymentSpecificationModel, 0, len(data.DeploymentSpecifications.Elements()))
		diag.Append(data.DeploymentSpecifications.ElementsAs(ctx, &deploymentSpecifications, false)...)

		if diag.HasError() {
			return utils.NvidiaCloudFunctionDeployment{}, true
		}

		deploymentSpecificationsOption := make([]utils.NvidiaCloudFunctionDeploymentSpecification, 0)
		for _, v := range deploymentSpecifications {
			var configuration interface{}
			if v.Configuration.ValueString() != "" {
				err := json.Unmarshal([]byte(v.Configuration.ValueString()), &configuration)

				if err != nil {
					diag.AddError(
						"Failed to create Cloud Function Deployment",
						err.Error(),
					)
				}

				if diag.HasError() {
					return utils.NvidiaCloudFunctionDeployment{}, true
				}
			}

			d := utils.NvidiaCloudFunctionDeploymentSpecification{
				Backend:               v.Backend.ValueString(),
				InstanceType:          v.InstanceType.ValueString(),
				Gpu:                   v.GpuType.ValueString(),
				MaxInstances:          int(v.MaxInstances.ValueInt64()),
				MinInstances:          int(v.MinInstances.ValueInt64()),
				MaxRequestConcurrency: int(v.MaxRequestConcurrency.ValueInt64()),
				Configuration:         configuration,
			}
			deploymentSpecificationsOption = append(deploymentSpecificationsOption, d)
		}

		var createNvidiaCloudFunctionDeploymentResponse, err = client.CreateNvidiaCloudFunctionDeployment(
			ctx, function.ID, function.VersionID,
			utils.CreateNvidiaCloudFunctionDeploymentRequest{
				DeploymentSpecifications: deploymentSpecificationsOption,
			},
		)

		if err != nil {
			diag.AddError(
				"Failed to create Cloud Function Deployment",
				err.Error(),
			)
		}

		if diag.HasError() {
			return utils.NvidiaCloudFunctionDeployment{}, true
		}

		err = client.WaitingDeploymentCompleted(ctx, function.ID, function.VersionID)

		if err != nil {
			diag.AddError(
				"Failed to create Cloud Function Deployment",
				err.Error(),
			)
		}

		if diag.HasError() {
			return utils.NvidiaCloudFunctionDeployment{}, true
		}

		functionDeployment = createNvidiaCloudFunctionDeploymentResponse.Deployment
	}
	return functionDeployment, false
}

func deploymentSpecificationsSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"configuration": schema.StringAttribute{
					MarkdownDescription: "Will be the json definition to overwrite the existing values.yaml file when deploying Helm-Based Functions",
					Optional:            true,
				},
				"backend": schema.StringAttribute{
					MarkdownDescription: "NVCF Backend.",
					Optional:            true,
				},
				"instance_type": schema.StringAttribute{
					MarkdownDescription: "NVCF Backend Instance Type.",
					Required:            true,
				},
				"gpu_type": schema.StringAttribute{
					MarkdownDescription: "GPU Type, GFN backend default is L40",
					Required:            true,
				},
				"max_instances": schema.Int64Attribute{
					MarkdownDescription: "Max Instances Count",
					Required:            true,
				},
				"min_instances": schema.Int64Attribute{
					MarkdownDescription: "Min Instances Count",
					Required:            true,
				},
				"max_request_concurrency": schema.Int64Attribute{
					MarkdownDescription: "Max Concurrency Count",
					Required:            true,
				},
			},
		},
		Optional: true,
		Computed: true,
		// The value will be auto-generated in NVCF API response when user using legacy health_uri field.
		PlanModifiers: []planmodifier.List{
			listplanmodifier.UseStateForUnknown(),
		},
	}
}

func resourcesSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: "Artifact name",
					Required:            true,
				},
				"version": schema.StringAttribute{
					MarkdownDescription: "Artifact version",
					Required:            true,
				},
				"uri": schema.StringAttribute{
					MarkdownDescription: "Artifact URI",
					Required:            true,
				},
			},
		},
		Optional: true,
	}
}

func modelsSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: "Artifact name",
					Required:            true,
				},
				"version": schema.StringAttribute{
					MarkdownDescription: "Artifact version",
					Required:            true,
				},
				"uri": schema.StringAttribute{
					MarkdownDescription: "Artifact URI",
					Required:            true,
				},
			},
		},
		Optional: true,
	}
}

func containerEnvironmentsSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"key": schema.StringAttribute{
					MarkdownDescription: "Container environment key",
					Required:            true,
				},
				"value": schema.StringAttribute{
					MarkdownDescription: "Container environment value",
					Required:            true,
				},
			},
		},
		Optional: true,
	}
}

func healthSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Computed: true,
		// The value will be auto-generated in NVCF API response when user using legacy health_uri field.
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"protocol": schema.StringAttribute{
				MarkdownDescription: "HTTP/gPRC protocol type for health endpoint",
				Required:            true,
			},
			"uri": schema.StringAttribute{
				MarkdownDescription: "Health endpoint for the container or the helmChart",
				Required:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "Port number where the health listener is running",
				Required:            true,
			},
			"timeout": schema.StringAttribute{
				MarkdownDescription: "ISO 8601 duration string in PnDTnHnMn.nS format",
				Required:            true,
			},
			"expected_status_code": schema.Int64Attribute{
				MarkdownDescription: "Expected return status code considered as successful",
				Required:            true,
			},
		},
	}
}

func (r *NvidiaCloudFunctionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Nvidia Cloud Function Resource",
		// TODO: Review PlanModifer
		// TODO: Need to clarify Computed means.
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Read-only Function ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"function_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Function ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"nca_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "NCA ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Function Version ID",
			},
			"function_name": schema.StringAttribute{
				MarkdownDescription: "Function name",
				Required:            true,
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
				Required:            true,
			},
			"health_uri": schema.StringAttribute{
				MarkdownDescription: "Service health endpoint Path. Default is \"/v2/health/ready\"",
				Optional:            true,
				Computed:            true,
				DeprecationMessage:  "The parameter is deprecated. Please replace it with `health`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"health":    healthSchema(),
			"resources": resourcesSchema(),
			"models":    modelsSchema(),
			"tags": schema.SetAttribute{
				MarkdownDescription: "Tags of the function.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the function",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"function_type": schema.StringAttribute{
				MarkdownDescription: "Optional function type, used to indicate a STREAMING function. Defaults is \"DEFAULT\".",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("DEFAULT"),
			},
			"api_body_format": schema.StringAttribute{
				MarkdownDescription: "API Body Format. Default is \"CUSTOM\"",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("CUSTOM"),
			},
			"deployment_specifications": deploymentSpecificationsSchema(),
			"keep_failed_resource": schema.BoolAttribute{
				MarkdownDescription: "Don't delete failed resource. Default is \"false\"",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Update: true,
			}),
		},
	}
}

func (r *NvidiaCloudFunctionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = ngcClient.NVCFClient()
}

func (r *NvidiaCloudFunctionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_function"
}

func (r *NvidiaCloudFunctionResource) createOrUpdateRequest(ctx context.Context, data NvidiaCloudFunctionResourceModel, diag *diag.Diagnostics) utils.CreateNvidiaCloudFunctionRequest {
	request := utils.CreateNvidiaCloudFunctionRequest{
		FunctionName:  data.FunctionName.ValueString(),
		InferencePort: int(data.InferencePort.ValueInt64()),
		InferenceUrl:  data.InferenceUrl.ValueString(),
		APIBodyFormat: data.APIBodyFormat.ValueString(),
		FunctionType:  data.FunctionType.ValueString(),
	}

	if !data.HelmChart.IsNull() && !data.HelmChart.IsUnknown() {
		request.HelmChart = data.HelmChart.ValueString()
	}

	if !data.HelmChartServiceName.IsNull() && !data.HelmChartServiceName.IsUnknown() {
		request.HelmChartServiceName = data.HelmChartServiceName.ValueString()
	}

	if !data.ContainerImage.IsNull() && !data.ContainerImage.IsUnknown() {
		request.ContainerImage = data.ContainerImage.ValueString()
	}

	if !data.ContainerArgs.IsNull() && !data.ContainerArgs.IsUnknown() {
		request.ContainerArgs = data.ContainerArgs.ValueString()
	}

	if !data.HealthUri.IsNull() && !data.HealthUri.IsUnknown() {
		request.HealthUri = data.HealthUri.ValueString()
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		request.Description = data.Description.ValueString()
	}

	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		var tags []string
		data.Tags.ElementsAs(ctx, &tags, true)
		request.Tags = tags
	}

	if !data.ContainerEnvironment.IsNull() && !data.ContainerEnvironment.IsUnknown() {
		containerEnvironments := make([]NvidiaCloudFunctionResourceContainerEnvironmentModel, 0)

		diag.Append(data.ContainerEnvironment.ElementsAs(ctx, &containerEnvironments, false)...)

		if diag.HasError() {
			return utils.CreateNvidiaCloudFunctionRequest{}
		}

		for _, v := range containerEnvironments {
			request.ContainerEnvironment = append(request.ContainerEnvironment, utils.NvidiaCloudFunctionContainerEnvironment{
				Key:   v.Key.ValueString(),
				Value: v.Value.ValueString(),
			})
		}
	}

	if !data.Health.IsNull() && !data.Health.IsUnknown() {
		health := &NvidiaCloudFunctionResourceHealthModel{}
		data.Health.As(ctx, health, basetypes.ObjectAsOptions{})
		request.Health = &utils.NvidiaCloudFunctionHealth{
			URI:                health.Uri.ValueString(),
			Port:               int(health.Port.ValueInt64()),
			Protocol:           health.Protocol.ValueString(),
			Timeout:            health.Timeout.ValueString(),
			ExpectedStatusCode: int(health.ExpectedStatusCode.ValueInt64()),
		}
	}

	if !data.Resources.IsNull() && !data.Resources.IsUnknown() {
		resources := make([]NvidiaCloudFunctionResourceResourceModel, 0)

		diag.Append(data.Resources.ElementsAs(ctx, &resources, false)...)

		if diag.HasError() {
			return utils.CreateNvidiaCloudFunctionRequest{}
		}

		for _, v := range resources {
			request.Resources = append(request.Resources, utils.NvidiaCloudFunctionResource{
				Name:    v.Name.ValueString(),
				Version: v.Version.ValueString(),
				URI:     v.Uri.ValueString(),
			})
		}
	}

	if !data.Models.IsNull() && !data.Models.IsUnknown() {
		models := make([]NvidiaCloudFunctionResourceModelModel, 0)

		diag.Append(data.Models.ElementsAs(ctx, &models, false)...)

		if diag.HasError() {
			return utils.CreateNvidiaCloudFunctionRequest{}
		}

		for _, v := range models {
			request.Models = append(request.Models, utils.NvidiaCloudFunctionModel{
				Name:    v.Name.ValueString(),
				Version: v.Version.ValueString(),
				URI:     v.Uri.ValueString(),
			})
		}
	}
	return request
}

func (r *NvidiaCloudFunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NvidiaCloudFunctionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := data.Timeouts.Create(ctx, DEFAULT_TIMEOUT_SEC*time.Second)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	request := r.createOrUpdateRequest(ctx, data, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	var createNvidiaCloudFunctionResponse, err = r.client.CreateNvidiaCloudFunction(
		ctx,
		data.FunctionID.ValueString(),
		request,
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create Cloud Function",
			err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	function := createNvidiaCloudFunctionResponse.Function

	if len(data.DeploymentSpecifications.Elements()) == 0 {
		r.updateNvidiaCloudFunctionResourceModel(ctx, &resp.Diagnostics, data.FunctionID, &data, &function, nil)
	} else {
		functionDeployment, hasError := createDeployment(ctx, data, &resp.Diagnostics, *r.client, function)

		if hasError {
			r.deleteFailedDeploymentVersion(ctx, data.KeepFailedResource.ValueBool(), function.ID, function.VersionID, &resp.Diagnostics)
			return
		}
		r.updateNvidiaCloudFunctionResourceModel(ctx, &resp.Diagnostics, data.FunctionID, &data, &function, &functionDeployment)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NvidiaCloudFunctionResource) deleteFailedDeploymentVersion(ctx context.Context, keepFailedResource bool, functionID string, versionID string, diag *diag.Diagnostics) {
	tflog.Error(ctx, "failed to deploy the new version.")
	if !keepFailedResource {
		err := r.client.DeleteNvidiaCloudFunctionVersion(ctx, functionID, versionID)
		if err != nil {
			diag.AddError(
				"Failed to delete failed Cloud Function deployment",
				err.Error(),
			)
			return
		}
		tflog.Info(ctx, "deleted the failed function deployment")
	}
}

func (r *NvidiaCloudFunctionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NvidiaCloudFunctionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var listNvidiaCloudFunctionVersionsResponse, err = r.client.ListNvidiaCloudFunctionVersions(ctx, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read Cloud Function versions",
			"Got unexpected result when reading Cloud Function",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	versionNotFound := true
	var functionVersion utils.NvidiaCloudFunctionInfo

	for _, f := range listNvidiaCloudFunctionVersionsResponse.Functions {
		if f.ID == data.Id.ValueString() && f.VersionID == data.VersionID.ValueString() {
			functionVersion = f
			versionNotFound = false
			break
		}
	}

	if versionNotFound {
		resp.Diagnostics.AddError("Version ID Not Found Error", fmt.Sprintf("Unable to find the target version ID %s", data.VersionID.ValueString()))
	}

	if resp.Diagnostics.HasError() {
		return
	}

	readNvidiaCloudFunctionDeploymentResponse, err := r.client.ReadNvidiaCloudFunctionDeployment(ctx, data.Id.ValueString(), data.VersionID.ValueString())

	if err != nil {
		// FIXME: extract error messsage to constants.
		if err.Error() != "failed to find function deployment" {
			resp.Diagnostics.AddError(
				"Failed to read Cloud Function deployment",
				err.Error(),
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	r.updateNvidiaCloudFunctionResourceModel(ctx, &resp.Diagnostics, data.FunctionID, &data, &functionVersion, &readNvidiaCloudFunctionDeploymentResponse.Deployment)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// TODO: Support deployment update, not recreate new function version.
func (r *NvidiaCloudFunctionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NvidiaCloudFunctionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, DEFAULT_TIMEOUT_SEC*time.Second)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	request := r.createOrUpdateRequest(ctx, plan, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	var createNvidiaCloudFunctionResponse, err = r.client.CreateNvidiaCloudFunction(ctx,
		plan.Id.ValueString(),
		request,
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update Cloud Function",
			err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	function := createNvidiaCloudFunctionResponse.Function

	if len(plan.DeploymentSpecifications.Elements()) == 0 {
		err = r.client.DeleteNvidiaCloudFunctionVersion(ctx, state.Id.ValueString(), state.VersionID.ValueString())
		// The case we still save state, since the deployment is disabled and user can delete the version manually.
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to delete Cloud Function version %s", plan.VersionID.ValueString()),
				err.Error(),
			)
		}
		r.updateNvidiaCloudFunctionResourceModel(ctx, &resp.Diagnostics, plan.FunctionID, &plan, &function, nil)
	} else {
		functionDeployment, hasError := createDeployment(ctx, plan, &resp.Diagnostics, *r.client, function)

		if hasError {
			r.deleteFailedDeploymentVersion(ctx, plan.KeepFailedResource.ValueBool(), function.ID, function.VersionID, &resp.Diagnostics)
			return
		}

		err = r.client.DeleteNvidiaCloudFunctionVersion(ctx, state.Id.ValueString(), state.VersionID.ValueString())
		// The case we still save state, since the deployment is disabled and user can delete the version manually.
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to delete Cloud Function version %s", plan.VersionID.ValueString()),
				err.Error(),
			)
		}
		r.updateNvidiaCloudFunctionResourceModel(ctx, &resp.Diagnostics, plan.FunctionID, &plan, &function, &functionDeployment)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NvidiaCloudFunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NvidiaCloudFunctionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	err := r.client.DeleteNvidiaCloudFunctionVersion(ctx, data.Id.ValueString(), data.VersionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to delete Cloud Function version %s", data.VersionID.ValueString()),
			err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *NvidiaCloudFunctionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: function_id,version_id. Got: %q", req.ID),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version_id"), idParts[1])...)
}
