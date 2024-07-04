package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab-master.nvidia.com/nvb/core/terraform-provider-ngc/internal/provider/utils"
)

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

type NvidiaCloudFunctionDeploymentSpecificationModel struct {
	GpuType               types.String `tfsdk:"gpu_type"`
	Backend               types.String `tfsdk:"backend"`
	MaxInstances          types.Int64  `tfsdk:"max_instances"`
	MinInstances          types.Int64  `tfsdk:"min_instances"`
	MaxRequestConcurrency types.Int64  `tfsdk:"max_request_concurrency"`
	Configuration         types.String `tfsdk:"configuration"`
	InstanceType          types.String `tfsdk:"instance_type"`
}

type NvidiaCloudFunctionResourceModel struct {
	Id                       types.String `tfsdk:"id"`
	FunctionId               types.String `tfsdk:"function_id"`
	VersionId                types.String `tfsdk:"version_id"`
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

func (r *NvidiaCloudFunctionResource) updateNvidiaCloudFunctionResourceModel(
	ctx context.Context, diag *diag.Diagnostics,
	userProvideFunctionId types.String,
	data *NvidiaCloudFunctionResourceModel,
	functionInfo *utils.NvidiaCloudFunctionInfo,
	functionDeployment *utils.NvidiaCloudFunctionDeployment) {

	data.Id = types.StringValue(functionInfo.ID)
	data.VersionId = types.StringValue(functionInfo.VersionID)
	data.FunctionName = types.StringValue(functionInfo.Name)
	data.FunctionId = userProvideFunctionId
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

func createDeployment(ctx context.Context, data NvidiaCloudFunctionResourceModel, diag *diag.Diagnostics, client utils.NVCFClient, function utils.NvidiaCloudFunctionInfo) (utils.NvidiaCloudFunctionDeployment, bool) {
	var functionDeployment utils.NvidiaCloudFunctionDeployment

	if !data.DeploymentSpecifications.IsNull() && len(data.DeploymentSpecifications.Elements()) > 0 {
		deploymentSpecifications := make([]NvidiaCloudFunctionDeploymentSpecificationModel, 0, len(data.DeploymentSpecifications.Elements()))
		diag.Append(data.DeploymentSpecifications.ElementsAs(ctx, &deploymentSpecifications, false)...)

		if diag.HasError() {
			return utils.NvidiaCloudFunctionDeployment{}, true
		}

		deploymentSpecificationsOption := make([]utils.NvidiaCloudFunctionDeploymentSpecification, 0)
		for _, v := range deploymentSpecifications {
			var configuration interface{}
			json.Unmarshal([]byte(v.Configuration.ValueString()), &configuration)

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
				"Failed to create NGC Cloud Function Deployment",
				err.Error(),
			)
		}

		if diag.HasError() {
			return utils.NvidiaCloudFunctionDeployment{}, true
		}

		err = client.WaitingDeploymentCompleted(ctx, function.ID, function.VersionID)

		if err != nil {
			diag.AddError(
				"Failed to create NGC Cloud Function Deployment",
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
			"helm_chart_uri": schema.StringAttribute{
				MarkdownDescription: "Helm chart registry uri",
				Optional:            true,
			},
			"helm_chart_service_name": schema.StringAttribute{
				MarkdownDescription: "Target service name",
				Optional:            true,
			},
			"helm_chart_service_port": schema.Int64Attribute{
				MarkdownDescription: "Target service port",
				Optional:            true,
			},
			"container_image_uri": schema.StringAttribute{
				MarkdownDescription: "Container image uri",
				Optional:            true,
			},
			"container_port": schema.Int64Attribute{
				MarkdownDescription: "Container port",
				Optional:            true,
			},
			"endpoint_path": schema.StringAttribute{
				MarkdownDescription: "Service endpoint Path. Default is \"/\"",
				Optional:            true,
			},
			"health_endpoint_path": schema.StringAttribute{
				MarkdownDescription: "Service health endpoint Path. Default is \"/v2/health/ready\"",
				Optional:            true,
			},
			"api_body_format": schema.StringAttribute{
				MarkdownDescription: "API Body Format. Default is \"CUSTOM\"",
				Optional:            true,
			},
			"deployment_specifications": deploymentSpecificationsSchema(),
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

func (r *NvidiaCloudFunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NvidiaCloudFunctionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var createNvidiaCloudFunctionResponse, err = r.client.CreateNvidiaCloudFunction(ctx, data.FunctionId.ValueString(),
		utils.CreateNvidiaCloudFunctionRequest{
			FunctionName:         data.FunctionName.ValueString(),
			HelmChartUri:         data.HelmChartUri.ValueString(),
			HelmChartServiceName: data.HelmChartServiceName.ValueString(),
			HelmChartServicePort: int(data.HelmChartServicePort.ValueInt64()),
			ContainerImageUri:    data.ContainerImageUri.ValueString(),
			ContainerPort:        int(data.ContainerPort.ValueInt64()),
			EndpointPath:         data.EndpointPath.ValueString(),
			HealthEndpointPath:   data.HealthEndpointPath.ValueString(),
			APIBodyFormat:        data.APIBodyFormat.ValueString(),
		})

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create NGC Cloud Function",
			"Got unexpected result when creating Cloud Function",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	function := createNvidiaCloudFunctionResponse.Function

	if len(data.DeploymentSpecifications.Elements()) == 0 {
		r.client.DeleteNvidiaCloudFunctionDeployment(ctx, data.Id.ValueString(), data.VersionId.ValueString())
		r.updateNvidiaCloudFunctionResourceModel(ctx, &resp.Diagnostics, data.FunctionId, &data, &function, nil)
	} else {
		functionDeployment, hasError := createDeployment(ctx, data, &resp.Diagnostics, *r.client, function)
		r.client.WaitingDeploymentCompleted(ctx, function.ID, function.VersionID)

		if hasError {
			return
		}
		r.updateNvidiaCloudFunctionResourceModel(ctx, &resp.Diagnostics, data.FunctionId, &data, &function, &functionDeployment)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NvidiaCloudFunctionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NvidiaCloudFunctionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var listNvidiaCloudFunctionVersionsResponse, err = r.client.ListNvidiaCloudFunctionVersions(ctx, utils.ListNvidiaCloudFunctionVersionsRequest{
		data.Id.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read NGC Cloud Function versions",
			"Got unexpected result when reading Cloud Function",
		)
	}

	versionNotFound := true
	var functionVersion utils.NvidiaCloudFunctionInfo

	for _, f := range listNvidiaCloudFunctionVersionsResponse.Functions {
		if f.ID == data.Id.ValueString() && f.VersionID == data.VersionId.ValueString() {
			functionVersion = f
			versionNotFound = false
			break
		}
	}

	if versionNotFound {
		resp.Diagnostics.AddError("Version ID Not Found Error", fmt.Sprintf("Unable to find the target version ID %s", data.VersionId.ValueString()))
	}

	if resp.Diagnostics.HasError() {
		return
	}

	readNvidiaCloudFunctionDeploymentResponse, err := r.client.ReadNvidiaCloudFunctionDeployment(ctx, data.Id.ValueString(), data.VersionId.ValueString())

	if err != nil {
		// FIXME: extract error messsage to constants.
		if err.Error() != "failed to find function deployment" {
			resp.Diagnostics.AddError(
				"Failed to read NGC Cloud Function deployment",
				err.Error(),
			)
		}
	}

	r.updateNvidiaCloudFunctionResourceModel(ctx, &resp.Diagnostics, data.FunctionId, &data, &functionVersion, &readNvidiaCloudFunctionDeploymentResponse.Deployment)

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

	var createNvidiaCloudFunctionResponse, err = r.client.CreateNvidiaCloudFunction(ctx,
		plan.Id.ValueString(),
		utils.CreateNvidiaCloudFunctionRequest{
			FunctionName:         plan.FunctionName.ValueString(),
			HelmChartUri:         plan.HelmChartUri.ValueString(),
			HelmChartServiceName: plan.HelmChartServiceName.ValueString(),
			HelmChartServicePort: int(plan.HelmChartServicePort.ValueInt64()),
			ContainerImageUri:    plan.ContainerImageUri.ValueString(),
			ContainerPort:        int(plan.ContainerPort.ValueInt64()),
			EndpointPath:         plan.EndpointPath.ValueString(),
			HealthEndpointPath:   plan.HealthEndpointPath.ValueString(),
			APIBodyFormat:        plan.APIBodyFormat.ValueString(),
		})

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update NGC Cloud Function",
			"Got unexpected result when updating Cloud Function",
		)
	}

	function := createNvidiaCloudFunctionResponse.Function

	if len(plan.DeploymentSpecifications.Elements()) == 0 {
		r.client.DeleteNvidiaCloudFunctionDeployment(ctx, plan.Id.ValueString(), plan.VersionId.ValueString())
		err = r.client.DeleteNvidiaCloudFunctionVersion(ctx, state.Id.ValueString(), state.VersionId.ValueString())
		// The case we still save state, since the deployment is disabled and user can delete the version manually.
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to delete NGC Cloud Function version %s", plan.VersionId.ValueString()),
				err.Error(),
			)
		}
		r.updateNvidiaCloudFunctionResourceModel(ctx, &resp.Diagnostics, plan.FunctionId, &plan, &function, nil)
	} else {
		functionDeployment, hasError := createDeployment(ctx, plan, &resp.Diagnostics, *r.client, function)
		r.client.WaitingDeploymentCompleted(ctx, function.ID, function.VersionID)

		if hasError {
			return
		}
		err = r.client.DeleteNvidiaCloudFunctionVersion(ctx, state.Id.ValueString(), state.VersionId.ValueString())
		// The case we still save state, since the deployment is disabled and user can delete the version manually.
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to delete NGC Cloud Function version %s", plan.VersionId.ValueString()),
				err.Error(),
			)
		}
		r.updateNvidiaCloudFunctionResourceModel(ctx, &resp.Diagnostics, plan.FunctionId, &plan, &function, &functionDeployment)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NvidiaCloudFunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NvidiaCloudFunctionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	err := r.client.DeleteNvidiaCloudFunctionVersion(ctx, data.Id.ValueString(), data.VersionId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to delete NGC Cloud Function version %s", data.VersionId.ValueString()),
			err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *NvidiaCloudFunctionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
