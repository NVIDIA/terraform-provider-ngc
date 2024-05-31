// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure NvidiaCloudFunctionProvider satisfies various provider interfaces.
var _ provider.Provider = &NvidiaCloudFunctionProvider{}
var _ provider.ProviderWithFunctions = &NvidiaCloudFunctionProvider{}

// NvidiaCloudFunctionProvider defines the provider implementation.
type NvidiaCloudFunctionProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// NvidiaCloudFunctionProviderModel describes the provider data model.
type NvidiaCloudFunctionProviderModel struct {
	NvidiaCloudFunctionEndpoint types.String `tfsdk:"nvidia_cloud_function_endpoint"`
	AuthTokenProviderEndpoint   types.String `tfsdk:"auth_token_provider_endpoint"`
	StarfleetClientId           types.String `tfsdk:"starfleet_client_id"`
	StarfleetClientSecret       types.String `tfsdk:"starfleet_client_secret"`
}

func (p *NvidiaCloudFunctionProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nvidia_cloud_function"
	resp.Version = p.version
}

func (p *NvidiaCloudFunctionProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"starfleet_client_id": schema.StringAttribute{
				MarkdownDescription: "Starfleet client ID",
				Optional:            true,
			},
			"starfleet_client_secret": schema.StringAttribute{
				MarkdownDescription: "Starfleet client secret",
				Optional:            true,
			},
			"nvidia_cloud_function_endpoint": schema.StringAttribute{
				MarkdownDescription: "NVIDIA Cloud Function API endpoint",
				Optional:            true,
			},
			"auth_token_provider_endpoint": schema.StringAttribute{
				MarkdownDescription: "Auth token provider endpoint",
				Optional:            true,
			},
		},
	}
}

func (p *NvidiaCloudFunctionProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	starfleetClientId := os.Getenv("STARFLEET_CLIENT_ID")
	starfleetClientSecret := os.Getenv("STARFLEET_CLIENT_SECRET")
	nvidiaCloudFunctionEndpoint := os.Getenv("NVIDIA_CLOUD_FUNCTION_ENDPOINT")
	authTokenProviderEndpoint := os.Getenv("AUTH_TOKEN_PROVIDER_ENDPOINT")

	var data NvidiaCloudFunctionProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Check configuration data, which should take precedence over
	// environment variable data, if found.
	if data.StarfleetClientId.ValueString() != "" {
		starfleetClientId = data.StarfleetClientId.ValueString()
	}

	if starfleetClientId == "" {
		resp.Diagnostics.AddError(
			"Missing Starfleet Client ID Configuration",
			"While configuring the provider, the Starfleet Client ID was not found in "+
				"the STARFLEET_CLIENT_ID environment variable or provider "+
				"configuration block starfleet_client_id attribute.",
		)
	}

	if data.StarfleetClientSecret.ValueString() != "" {
		starfleetClientSecret = data.StarfleetClientSecret.ValueString()
	}

	if starfleetClientSecret == "" {
		resp.Diagnostics.AddError(
			"Missing Starfleet Client Secret Configuration",
			"While configuring the provider, the Starfleet Client Secret was not found in "+
				"the STARFLEET_CLIENT_SECRET environment variable or provider "+
				"configuration block starfleet_client_secret attribute.",
		)
	}

	if data.NvidiaCloudFunctionEndpoint.ValueString() != "" {
		nvidiaCloudFunctionEndpoint = data.NvidiaCloudFunctionEndpoint.ValueString()
	}

	if nvidiaCloudFunctionEndpoint == "" {
		nvidiaCloudFunctionEndpoint = "https://api.nvcf.nvidia.com"
	}

	if data.AuthTokenProviderEndpoint.ValueString() != "" {
		authTokenProviderEndpoint = data.AuthTokenProviderEndpoint.ValueString()
	}

	if authTokenProviderEndpoint == "" {
		authTokenProviderEndpoint = "https://tbyyhdy8-opimayg5nq78mx1wblbi8enaifkmlqrm8m.ssa.nvidia.com"
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (p *NvidiaCloudFunctionProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNvidiaCloudFunctionResource,
	}
}

func (p *NvidiaCloudFunctionProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
	}
}

func (p *NvidiaCloudFunctionProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		NewExampleFunction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NvidiaCloudFunctionProvider{
			version: version,
		}
	}
}
