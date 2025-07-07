package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NvidiaCloudFunctionTelemetryResourceSecretModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type NvidiaCloudFunctionTelemetryResourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Endpoint  types.String `tfsdk:"endpoint"`
	Protocol  types.String `tfsdk:"protocol"`
	Provider  types.String `tfsdk:"telemetry_provider"`
	Types     types.Set    `tfsdk:"types"`
	Secret    types.Object `tfsdk:"secret"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (m *NvidiaCloudFunctionTelemetryResourceSecretModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":  types.StringType,
		"value": types.StringType,
	}
}
