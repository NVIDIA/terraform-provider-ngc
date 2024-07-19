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
	FunctionID               types.String `tfsdk:"function_id"`
	VersionID                types.String `tfsdk:"version_id"`
	NcaId                    types.String `tfsdk:"nca_id"`
	FunctionName             types.String `tfsdk:"function_name"`
	HelmChartUri             types.String `tfsdk:"helm_chart_uri"`
	HelmChartServiceName     types.String `tfsdk:"helm_chart_service_name"`
	HelmChartServicePort     types.Int64  `tfsdk:"helm_chart_service_port"`
	ContainerImageUri        types.String `tfsdk:"container_image_uri"`
	ContainerPort            types.Int64  `tfsdk:"container_port"`
	EndpointPath             types.String `tfsdk:"endpoint_path"`
	HealthEndpointPath       types.String `tfsdk:"health_endpoint_path"`
	APIBodyFormat            types.String `tfsdk:"api_body_format"`
	DeploymentSpecifications types.List   `tfsdk:"deployment_specifications"`
}

func (d *NvidiaCloudFunctionDataSource) updateNvidiaCloudFunctionDataSourceModel(
	ctx context.Context, diag *diag.Diagnostics,
	data *NvidiaCloudFunctionDataSourceModel,
	functionInfo *utils.NvidiaCloudFunctionInfo,
	functionDeployment *utils.NvidiaCloudFunctionDeployment) {
	data.VersionID = types.StringValue(functionInfo.VersionID)
	data.FunctionName = types.StringValue(functionInfo.Name)
	data.FunctionID = types.StringValue(functionInfo.ID)
	data.APIBodyFormat = types.StringValue(functionInfo.APIBodyFormat)
	data.NcaId = types.StringValue(functionInfo.NcaID)
	data.APIBodyFormat = types.StringValue(functionInfo.APIBodyFormat)
	data.EndpointPath = types.StringValue(functionInfo.InferenceURL)
	data.HealthEndpointPath = types.StringValue(functionInfo.HealthURI)

	if functionInfo.HelmChart != "" {
		data.HelmChartServicePort = types.Int64Value(int64(functionInfo.InferencePort))
		data.HelmChartServiceName = types.StringValue(functionInfo.HelmServiceName)
		data.HelmChartUri = types.StringValue(functionInfo.HelmChart)
	} else {
		data.ContainerPort = types.Int64Value(int64(functionInfo.InferencePort))
		data.ContainerImageUri = types.StringValue(functionInfo.ContainerImage)
	}

	if functionDeployment != nil {
		deploymentSpecifications := make([]NvidiaCloudFunctionDeploymentSpecificationModel, 0)

		for _, v := range functionDeployment.DeploymentSpecifications {
			deploymentSpecification := NvidiaCloudFunctionDeploymentSpecificationModel{
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
		deploymentSpecificationsSetType, deploymentSpecificationsSetTypeDiag := types.ListValueFrom(ctx, deploymentSpecificationsSchema().NestedObject.Type(), deploymentSpecifications)
		diag.Append(deploymentSpecificationsSetTypeDiag...)
		data.DeploymentSpecifications = deploymentSpecificationsSetType
	}
}

func (d *NvidiaCloudFunctionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_function"
}

func (d *NvidiaCloudFunctionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example data source",

		Attributes: map[string]schema.Attribute{
			"function_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Function ID",
			},
			"nca_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "NCA ID",
			},
			"version_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Function Version ID",
			},
			"function_name": schema.StringAttribute{
				MarkdownDescription: "Function name",
				Computed:            true,
			},
			"helm_chart_uri": schema.StringAttribute{
				MarkdownDescription: "Helm chart registry uri",
				Computed:            true,
			},
			"helm_chart_service_name": schema.StringAttribute{
				MarkdownDescription: "Target service name",
				Computed:            true,
			},
			"helm_chart_service_port": schema.Int64Attribute{
				MarkdownDescription: "Target service port",
				Computed:            true,
			},
			"container_image_uri": schema.StringAttribute{
				MarkdownDescription: "Container image uri",
				Computed:            true,
			},
			"container_port": schema.Int64Attribute{
				MarkdownDescription: "Container port",
				Computed:            true,
			},
			"endpoint_path": schema.StringAttribute{
				MarkdownDescription: "Service endpoint Path. Default is \"/\"",
				Computed:            true,
			},
			"health_endpoint_path": schema.StringAttribute{
				MarkdownDescription: "Service health endpoint Path. Default is \"/v2/health/ready\"",
				Computed:            true,
			},
			"api_body_format": schema.StringAttribute{
				MarkdownDescription: "API Body Format. Default is \"CUSTOM\"",
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
	}

	if resp.Diagnostics.HasError() {
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
		}
	}

	d.updateNvidiaCloudFunctionDataSourceModel(ctx, &resp.Diagnostics, &data, &functionVersion, &readNvidiaCloudFunctionDeploymentResponse.Deployment)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
