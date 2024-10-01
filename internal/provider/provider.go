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
	"os"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab-master.nvidia.com/nvb/core/terraform-provider-ngc/internal/provider/utils"
)

// Ensure NgcProvider satisfies various provider interfaces.
var _ provider.Provider = &NgcProvider{}
var _ provider.ProviderWithFunctions = &NgcProvider{}

// NgcProvider defines the provider implementation.
type NgcProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// NgcProviderModel describes the provider data model.
type NgcProviderModel struct {
	NgcEndpoint types.String `tfsdk:"ngc_endpoint"`
	NgcApiKey   types.String `tfsdk:"ngc_api_key"`
	NgcOrg      types.String `tfsdk:"ngc_org"`
	NgcTeam     types.String `tfsdk:"ngc_team"`
}

func (p *NgcProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ngc"
	resp.Version = p.version
}

func (p *NgcProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"ngc_endpoint": schema.StringAttribute{
				MarkdownDescription: "NGC API endpoint",
				Optional:            true,
			},
			"ngc_api_key": schema.StringAttribute{
				MarkdownDescription: "NGC Personal Token with `Cloud Function` permission",
				Optional:            true,
				Sensitive:           true,
			},
			"ngc_org": schema.StringAttribute{
				MarkdownDescription: "NGC Org Name.",
				Optional:            true,
			},
			"ngc_team": schema.StringAttribute{
				MarkdownDescription: "NGC Team Name",
				Optional:            true,
			},
		},
	}
}

func (p *NgcProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	ngcEndpoint := os.Getenv("NGC_ENDPOINT")
	ngcApiKey := os.Getenv("NGC_API_KEY")
	ngcOrg := os.Getenv("NGC_ORG")
	ngcTeam := os.Getenv("NGC_TEAM")

	var data NgcProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Check configuration data, which should take precedence over
	// environment variable data, if found.
	if data.NgcApiKey.ValueString() != "" {
		ngcApiKey = data.NgcApiKey.ValueString()
	}

	if ngcApiKey == "" {
		resp.Diagnostics.AddError(
			"Missing NGC_API_KEY Configuration",
			"While configuring the provider, the NGC personal key was not found in "+
				"the NGC_API_KEY environment variable or provider "+
				"configuration block ngc_api_key attribute.",
		)
	}

	if data.NgcOrg.ValueString() != "" {
		ngcOrg = data.NgcOrg.ValueString()
	}

	if ngcOrg == "" {
		resp.Diagnostics.AddError(
			"Missing NGC_ORG Configuration",
			"While configuring the provider, the NGC Org Name was not found in "+
				"the NGC_ORG environment variable or provider "+
				"configuration block ngc_org attribute.",
		)
	}

	if data.NgcTeam.ValueString() != "" {
		ngcTeam = data.NgcTeam.ValueString()
	}

	if data.NgcEndpoint.ValueString() != "" {
		ngcEndpoint = data.NgcEndpoint.ValueString()
	}

	if ngcEndpoint == "" {
		ngcEndpoint = "https://api.ngc.nvidia.com"
	}

	if resp.Diagnostics.HasError() {
		return
	}

	httpClient := cleanhttp.DefaultPooledClient()

	client := &utils.NGCClient{
		NgcEndpoint: ngcEndpoint,
		NgcApiKey:   ngcApiKey,
		NgcOrg:      ngcOrg,
		NgcTeam:     ngcTeam,
		HttpClient:  httpClient,
	}
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *NgcProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNvidiaCloudFunctionResource,
	}
}

func (p *NgcProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewNvidiaCloudFunctionDataSource,
	}
}

func (p *NgcProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NgcProvider{
			version: version,
		}
	}
}
