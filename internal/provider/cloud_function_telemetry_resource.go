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

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gitlab-master.nvidia.com/nvb/core/terraform-provider-ngc/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &NvidiaCloudFunctionTelemetryResource{}
var _ resource.ResourceWithImportState = &NvidiaCloudFunctionTelemetryResource{}

func NewNvidiaCloudFunctionTelemetryResource() resource.Resource {
	return &NvidiaCloudFunctionTelemetryResource{}
}

// NvidiaCloudFunctionTelemetryResource defines the resource implementation.
type NvidiaCloudFunctionTelemetryResource struct {
	client *utils.NVCFClient
}

func (r *NvidiaCloudFunctionTelemetryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_function_telemetry"
}

func (r *NvidiaCloudFunctionTelemetryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "NVIDIA Cloud Function Telemetry Resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique telemetry ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Telemetry name, will be same as the secret name",
			},
			"endpoint": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "URL for the telemetry endpoint",
			},
			"protocol": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Protocol used for communication (HTTP or GRPC)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"telemetry_provider": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Telemetry provider (PROMETHEUS, GRAFANA_CLOUD, SPLUNK, DATADOG, SERVICENOW, KRATOS, KRATOS_THANOS, AZURE_MONITOR)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"types": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Set of telemetry data types (LOGS, METRICS, TRACES)",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"secret": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Secret configuration for the telemetry",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Secret name",
					},
					"value": schema.StringAttribute{
						Required:            true,
						Sensitive:           true,
						MarkdownDescription: "Secret value",
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Telemetry creation timestamp",
			},
		},
	}
}

func (r *NvidiaCloudFunctionTelemetryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NvidiaCloudFunctionTelemetryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NvidiaCloudFunctionTelemetryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Extract types from the set
	var types []string
	resp.Diagnostics.Append(data.Types.ElementsAs(ctx, &types, false)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Extract secret from the object
	var secret NvidiaCloudFunctionTelemetryResourceSecretModel
	resp.Diagnostics.Append(data.Secret.As(ctx, &secret, basetypes.ObjectAsOptions{})...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create the telemetry request
	telemetryRequest := utils.CreateNvidiaCloudFunctionTelemetryRequest{
		Protocol: data.Protocol.ValueString(),
		Provider: data.Provider.ValueString(),
		Types:    types,
		Secret: utils.NvidiaCloudFunctionTelemetrySecret{
			Name:  secret.Name.ValueString(),
			Value: secret.Value.ValueString(),
		},
	}

	if secret.Value.ValueString() != "" {
		var secretValue interface{}
		err := json.Unmarshal([]byte(secret.Value.ValueString()), &secretValue)

		// When the input is not a valid json, we will put it as string directly.
		if err != nil {
			telemetryRequest.Secret = utils.NvidiaCloudFunctionTelemetrySecret{
				Name:  secret.Name.ValueString(),
				Value: secret.Value.ValueString(),
			}
		} else {
			telemetryRequest.Secret = utils.NvidiaCloudFunctionTelemetrySecret{
				Name:  secret.Name.ValueString(),
				Value: secretValue,
			}
		}
	}

	if !data.Endpoint.IsNull() && !data.Endpoint.IsUnknown() {
		telemetryRequest.Endpoint = data.Endpoint.ValueString()
	}

	// Create the telemetry
	telemetryResponse, err := r.client.CreateTelemetry(ctx, telemetryRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create telemetry",
			err.Error(),
		)
		return
	}

	// Update the model with the response
	r.updateTelemetryResourceModel(ctx, &data, &telemetryResponse.Telemetry)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NvidiaCloudFunctionTelemetryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NvidiaCloudFunctionTelemetryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the telemetry
	telemetryResponse, err := r.client.GetTelemetry(ctx, data.Id.ValueString())
	if err != nil {
		// Check if the error indicates that the resource was not found
		if strings.Contains(err.Error(), "Not found") || strings.Contains(err.Error(), "404") {
			// Resource does not exist anymore, remove from state
			tflog.Warn(ctx, fmt.Sprintf("Telemetry %s no longer exists, removing from state", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}

		// For other errors, report them as usual
		resp.Diagnostics.AddError(
			"Failed to get telemetry",
			err.Error(),
		)
		return
	}

	// Update the model with the response
	r.updateTelemetryResourceModel(ctx, &data, &telemetryResponse.Telemetry)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NvidiaCloudFunctionTelemetryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported - implementation error.",
		"Telemetry APIs do not support updates. You should make sure all the changes will trigger force-replaced.",
	)
}

func (r *NvidiaCloudFunctionTelemetryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NvidiaCloudFunctionTelemetryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the telemetry
	err := r.client.DeleteTelemetry(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to delete telemetry %s", data.Id.ValueString()),
			err.Error(),
		)
		return
	}
}

func (r *NvidiaCloudFunctionTelemetryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by telemetry ID
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// updateTelemetryResourceModel updates the Terraform model with data from the API response.
func (r *NvidiaCloudFunctionTelemetryResource) updateTelemetryResourceModel(ctx context.Context, data *NvidiaCloudFunctionTelemetryResourceModel, telemetry *utils.NvidiaCloudFunctionTelemetry) {
	data.Id = types.StringValue(telemetry.TelemetryId)
	data.Protocol = types.StringValue(telemetry.Protocol)
	data.Provider = types.StringValue(telemetry.Provider)
	data.CreatedAt = types.StringValue(telemetry.CreatedAt.Format("2006-01-02T15:04:05Z"))
	data.Name = types.StringValue(telemetry.Name)

	if telemetry.Endpoint != "" {
		data.Endpoint = types.StringValue(telemetry.Endpoint)
	}

	// Convert types to set
	if telemetry.Types != nil {
		typesSet, diag := types.SetValueFrom(ctx, types.StringType, telemetry.Types)
		if diag.HasError() {
			return
		}
		data.Types = typesSet
	}

	// Note: We don't update Secret from response since it's sensitive information
	// and won't be returned in the response. We keep the original secret data
	// from the Terraform configuration.
}
