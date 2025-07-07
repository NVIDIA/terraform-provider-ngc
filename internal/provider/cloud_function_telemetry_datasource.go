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
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab-master.nvidia.com/nvb/core/terraform-provider-ngc/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &NvidiaCloudFunctionTelemetryDataSource{}

func NewNvidiaCloudFunctionTelemetryDataSource() datasource.DataSource {
	return &NvidiaCloudFunctionTelemetryDataSource{}
}

// NvidiaCloudFunctionTelemetryDataSource defines the data source implementation.
type NvidiaCloudFunctionTelemetryDataSource struct {
	client *utils.NVCFClient
}

// NvidiaCloudFunctionTelemetryDataSourceModel describes the data source data model.
type NvidiaCloudFunctionTelemetryDataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Endpoint  types.String `tfsdk:"endpoint"`
	Protocol  types.String `tfsdk:"protocol"`
	Provider  types.String `tfsdk:"telemetry_provider"`
	Types     types.Set    `tfsdk:"types"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (d *NvidiaCloudFunctionTelemetryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_function_telemetry"
}

func (d *NvidiaCloudFunctionTelemetryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "NVIDIA Cloud Function Telemetry Data Source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique telemetry ID",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Telemetry name",
			},
			"endpoint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL for the telemetry endpoint",
			},
			"protocol": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Protocol used for communication (HTTP or GRPC)",
			},
			"telemetry_provider": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Telemetry provider (PROMETHEUS, GRAFANA_CLOUD, SPLUNK, DATADOG, SERVICENOW, KRATOS, KRATOS_THANOS, TIMESTREAM, VICTORIAMETRICS, AZURE_MONITOR)",
			},
			"types": schema.SetAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "Set of telemetry data types (LOGS, METRICS, TRACES)",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Telemetry creation timestamp",
			},
		},
	}
}

func (d *NvidiaCloudFunctionTelemetryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NvidiaCloudFunctionTelemetryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NvidiaCloudFunctionTelemetryDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the telemetry using the client
	telemetryResponse, err := d.client.GetTelemetry(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read telemetry",
			err.Error(),
		)
		return
	}

	// Update the model with the response
	d.updateTelemetryDataSourceModel(ctx, &data, &telemetryResponse.Telemetry)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// updateTelemetryDataSourceModel updates the Terraform model with data from the API response
func (d *NvidiaCloudFunctionTelemetryDataSource) updateTelemetryDataSourceModel(ctx context.Context, data *NvidiaCloudFunctionTelemetryDataSourceModel, telemetry *utils.NvidiaCloudFunctionTelemetry) {
	data.Id = types.StringValue(telemetry.TelemetryId)
	data.Name = types.StringValue(telemetry.Name)
	data.Protocol = types.StringValue(telemetry.Protocol)
	data.Provider = types.StringValue(telemetry.Provider)
	data.CreatedAt = types.StringValue(telemetry.CreatedAt.Format("2006-01-02T15:04:05Z"))

	if telemetry.Endpoint != "" {
		data.Endpoint = types.StringValue(telemetry.Endpoint)
	}

	// Convert types to set
	if telemetry.Types != nil {
		typesSet, _ := types.SetValueFrom(ctx, types.StringType, telemetry.Types)
		data.Types = typesSet
	}
}
