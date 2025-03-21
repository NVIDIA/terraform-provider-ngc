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
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
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

func (r *NvidiaCloudFunctionResource) updateNvidiaCloudFunctionResourceModelBaseOnResponse(
	ctx context.Context, diag *diag.Diagnostics,
	data *NvidiaCloudFunctionResourceModel,
	functionInfo *utils.NvidiaCloudFunctionInfo,
) {
	data.Id = types.StringValue(functionInfo.ID)
	data.VersionID = types.StringValue(functionInfo.VersionID)
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
}

func authorizeAccountToInvokeFunction(
	ctx context.Context,
	functionID string,
	versionID string,
	authorizePartiesRawData basetypes.SetValue,
	diag *diag.Diagnostics,
	client utils.NVCFClient,
) utils.AuthorizeAccountsToInvokeFunctionResponse {
	if !authorizePartiesRawData.IsNull() && len(authorizePartiesRawData.Elements()) > 0 {
		authorizePartiesInTerraformModel := make([]NvidiaCloudFunctionResourceAuthorizedPartyModel, 0, len(authorizePartiesRawData.Elements()))
		diag.Append(authorizePartiesRawData.ElementsAs(ctx, &authorizePartiesInTerraformModel, false)...)

		if diag.HasError() {
			return utils.AuthorizeAccountsToInvokeFunctionResponse{}
		}

		authorizeParties := make([]utils.AuthorizedParty, 0)

		for _, v := range authorizePartiesInTerraformModel {
			authorizeParties = append(authorizeParties, utils.AuthorizedParty{
				NcaID: v.NcaID.ValueString(),
			})
		}
		var authorizeAccountsToInvokeFunctionResponse, err = client.AuthorizeAccountsToInvokeFunction(
			ctx, functionID, versionID,
			utils.AuthorizeAccountsToInvokeFunctionRequest{
				AuthorizedParties: authorizeParties,
			},
		)

		if err != nil {
			diag.AddError(
				"Failed to authorize additional accounts to invoke function",
				err.Error(),
			)
			return utils.AuthorizeAccountsToInvokeFunctionResponse{}
		}
		return *authorizeAccountsToInvokeFunctionResponse
	}
	return utils.AuthorizeAccountsToInvokeFunctionResponse{}
}

func normalizeJSON(s string) (string, error) {
	if s == "" {
		return s, nil
	}
	var j interface{}
	if err := json.Unmarshal([]byte(s), &j); err != nil {
		return "", err
	}
	normalized, err := json.Marshal(j)
	if err != nil {
		return "", err
	}
	return string(normalized), nil
}

func deploymentSpecificationsSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"configuration": schema.StringAttribute{
					MarkdownDescription: "Will be the json definition to overwrite the existing values.yaml file when deploying Helm-Based Functions",
					Optional:            true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplace(),
					},
				},
				"backend": schema.StringAttribute{
					MarkdownDescription: "NVCF Backend.",
					Optional:            true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplace(),
					},
				},
				"instance_type": schema.StringAttribute{
					MarkdownDescription: "NVCF Backend Instance Type.",
					Required:            true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplace(),
					},
				},
				"gpu_type": schema.StringAttribute{
					MarkdownDescription: "GPU Type, GFN backend default is L40",
					Required:            true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplace(),
					},
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
		PlanModifiers: []planmodifier.Set{
			setplanmodifier.UseStateForUnknown(),
		},
	}
}

func resourcesSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
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
		PlanModifiers: []planmodifier.Set{
			setplanmodifier.RequiresReplace(),
		},
	}
}

func modelsSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
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
		PlanModifiers: []planmodifier.Set{
			setplanmodifier.RequiresReplace(),
		},
	}
}

func containerEnvironmentsSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
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
		PlanModifiers: []planmodifier.Set{
			setplanmodifier.RequiresReplace(),
		},
	}
}

func healthSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Computed: true,
		// The value will be auto-generated in NVCF API response when user using legacy health_uri field.
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
			objectplanmodifier.RequiresReplace(),
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

func secretsSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: "Secret name",
					Required:            true,
				},
				"value": schema.StringAttribute{
					MarkdownDescription: "Secret value. Must be a string or json node.",
					Required:            true,
					Sensitive:           true,
				},
			},
		},
		Optional: true,
	}
}

func authorizedPartiesSchema() schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"nca_id": schema.StringAttribute{
					MarkdownDescription: "NVIDIA Cloud Account authorized to invoke the function",
					Required:            true,
				},
			},
		},
		MarkdownDescription: "Associated authorized parties for a specific version of a function",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"function_name": schema.StringAttribute{
				MarkdownDescription: "Function name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"helm_chart": schema.StringAttribute{
				MarkdownDescription: "Helm chart registry uri",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"helm_chart_service_name": schema.StringAttribute{
				MarkdownDescription: "Target service name",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"inference_port": schema.Int64Attribute{
				MarkdownDescription: "Target port, will be service port or container port base on function-based",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"container_image": schema.StringAttribute{
				MarkdownDescription: "Container image uri",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"container_environment": containerEnvironmentsSchema(),
			"container_args": schema.StringAttribute{
				MarkdownDescription: "Args to be passed when launching the container",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"inference_url": schema.StringAttribute{
				MarkdownDescription: "Service endpoint Path.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"api_body_format": schema.StringAttribute{
				MarkdownDescription: "API Body Format. Default is \"CUSTOM\"",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("CUSTOM"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_specifications": deploymentSpecificationsSchema(),
			"secrets":                   secretsSchema(),
			"authorized_parties":        authorizedPartiesSchema(),
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

//nolint:gocyclo
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

	if !data.Secrets.IsNull() && !data.Secrets.IsUnknown() {
		secrets := make([]NvidiaCloudFunctionResourceSecretModel, 0)
		diag.Append(data.Secrets.ElementsAs(ctx, &secrets, false)...)

		if diag.HasError() {
			return utils.CreateNvidiaCloudFunctionRequest{}
		}

		for _, v := range secrets {
			var secretValue interface{}
			if v.Value.ValueString() != "" {
				err := json.Unmarshal([]byte(v.Value.ValueString()), &secretValue)

				// When the input is not a valid json, we will put it as string directly.
				if err != nil {
					request.Secrets = append(request.Secrets, utils.NvidiaCloudFunctionSecret{
						Name:  v.Name.ValueString(),
						Value: v.Value.ValueString(),
					})
				} else {
					request.Secrets = append(request.Secrets, utils.NvidiaCloudFunctionSecret{
						Name:  v.Name.ValueString(),
						Value: secretValue,
					})
				}
			}
		}
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

	authorizeAccountToInvokeFunction(ctx, function.ID, function.VersionID, data.AuthorizedParties, &resp.Diagnostics, *r.client)

	if resp.Diagnostics.HasError() {
		return
	}

	if len(data.DeploymentSpecifications.Elements()) == 0 {
		r.updateNvidiaCloudFunctionResourceModelBaseOnResponse(ctx, &resp.Diagnostics, &data, &function)
	} else {
		r.createDeployment(ctx, data, &resp.Diagnostics, function)

		if resp.Diagnostics.HasError() {
			r.deleteFailedDeploymentVersion(ctx, data.KeepFailedResource.ValueBool(), function.ID, function.VersionID, &resp.Diagnostics)
			return
		}
		r.updateNvidiaCloudFunctionResourceModelBaseOnResponse(ctx, &resp.Diagnostics, &data, &function)
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

	var getFunctionVersionResponse, err = r.client.GetNvidiaCloudFunctionVersion(ctx, data.Id.ValueString(), data.VersionID.ValueString())

	if err != nil {
		// Check if the error indicates that the resource was not found
		if strings.Contains(err.Error(), "Not found") {
			// Resource does not exist anymore, remove from state
			tflog.Warn(ctx, fmt.Sprintf("Cloud Function version %s/%s no longer exists, removing from state", data.Id.ValueString(), data.VersionID.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}

		// For other errors, report them as usual
		resp.Diagnostics.AddError(
			"Failed to Get Cloud Function version",
			err.Error(),
		)
		return
	}

	_, err = r.client.ReadNvidiaCloudFunctionDeployment(ctx, data.Id.ValueString(), data.VersionID.ValueString())

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

	_, err = r.client.GetFunctionAuthorization(ctx, data.Id.ValueString(), data.VersionID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Cloud Function authorization",
			err.Error(),
		)
		return
	}

	r.updateNvidiaCloudFunctionResourceModelBaseOnResponse(ctx, &resp.Diagnostics, &data, &getFunctionVersionResponse.Function)

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

	// We don't need to clean up the old function authorizeParties since we will create new functions to replace old one every time.
	// However if we want to support update without function recreation, we need to handle the case for remove all authorized accounts
	// (i.e, the AuthorizedParties become empty)
	authorizeAccountToInvokeFunction(ctx, state.Id.String(), state.VersionID.String(), plan.AuthorizedParties, &resp.Diagnostics, *r.client)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetNvidiaCloudFunctionVersion(ctx, state.Id.ValueString(), state.VersionID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Cloud Function",
			err.Error(),
		)
	}

	if len(plan.DeploymentSpecifications.Elements()) == 0 {
		_, err := r.client.DeleteNvidiaCloudFunctionDeployment(ctx, state.Id.ValueString(), state.VersionID.ValueString())
		// The case we still save state, since the deployment is disabled and user can delete the version manually.
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to delete Cloud Function Deployment %s", plan.VersionID.ValueString()),
				err.Error(),
			)
		}
	} else {
		r.updateDeployment(ctx, plan, &resp.Diagnostics)

		if resp.Diagnostics.HasError() {
			return
		}
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

func (r *NvidiaCloudFunctionResource) prepareDeploymentSpecifications(
	ctx context.Context,
	data NvidiaCloudFunctionResourceModel,
	diag *diag.Diagnostics,
) []utils.NvidiaCloudFunctionDeploymentSpecification {
	if data.DeploymentSpecifications.IsNull() || len(data.DeploymentSpecifications.Elements()) == 0 {
		return nil
	}

	deploymentSpecifications := make([]NvidiaCloudFunctionResourceDeploymentSpecificationModel, 0, len(data.DeploymentSpecifications.Elements()))
	diag.Append(data.DeploymentSpecifications.ElementsAs(ctx, &deploymentSpecifications, false)...)

	if diag.HasError() {
		return nil
	}

	deploymentSpecificationsOption := make([]utils.NvidiaCloudFunctionDeploymentSpecification, 0)
	for _, v := range deploymentSpecifications {
		var configuration interface{}
		if v.Configuration.ValueString() != "" {
			err := json.Unmarshal([]byte(v.Configuration.ValueString()), &configuration)

			if err != nil {
				diag.AddError(
					"Failed to parse deployment configuration",
					err.Error(),
				)
				return nil
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

	return deploymentSpecificationsOption
}

func (r *NvidiaCloudFunctionResource) createDeployment(ctx context.Context, data NvidiaCloudFunctionResourceModel, diag *diag.Diagnostics, function utils.NvidiaCloudFunctionInfo) utils.NvidiaCloudFunctionDeployment {
	var functionDeployment utils.NvidiaCloudFunctionDeployment

	deploymentSpecificationsOption := r.prepareDeploymentSpecifications(ctx, data, diag)
	if diag.HasError() || deploymentSpecificationsOption == nil {
		return functionDeployment
	}

	createNvidiaCloudFunctionDeploymentResponse, err := r.client.CreateNvidiaCloudFunctionDeployment(
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
		return functionDeployment
	}

	err = r.client.WaitingDeploymentCompleted(ctx, function.ID, function.VersionID)
	if err != nil {
		diag.AddError(
			"Failed to create Cloud Function Deployment",
			err.Error(),
		)
		return functionDeployment
	}

	return createNvidiaCloudFunctionDeploymentResponse.Deployment
}

func (r *NvidiaCloudFunctionResource) updateDeployment(ctx context.Context, data NvidiaCloudFunctionResourceModel, diag *diag.Diagnostics) utils.NvidiaCloudFunctionDeployment {
	var functionDeployment utils.NvidiaCloudFunctionDeployment

	deploymentSpecificationsOption := r.prepareDeploymentSpecifications(ctx, data, diag)
	if diag.HasError() || deploymentSpecificationsOption == nil {
		return functionDeployment
	}

	updateNvidiaCloudFunctionDeploymentResponse, err := r.client.UpdateNvidiaCloudFunctionDeployment(
		ctx, data.Id.ValueString(), data.VersionID.ValueString(),
		utils.UpdateNvidiaCloudFunctionDeploymentRequest{
			DeploymentSpecifications: deploymentSpecificationsOption,
		},
	)

	if err != nil {
		diag.AddError(
			"Failed to update Cloud Function Deployment",
			err.Error(),
		)
		return functionDeployment
	}

	err = r.client.WaitingDeploymentCompleted(ctx, data.Id.ValueString(), data.VersionID.ValueString())
	if err != nil {
		diag.AddError(
			"Failed to update Cloud Function Deployment",
			err.Error(),
		)
		return functionDeployment
	}

	return updateNvidiaCloudFunctionDeploymentResponse.Deployment
}
