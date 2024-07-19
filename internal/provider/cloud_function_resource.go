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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	Id                       types.String   `tfsdk:"id"`
	FunctionID               types.String   `tfsdk:"function_id"`
	VersionID                types.String   `tfsdk:"version_id"`
	NcaId                    types.String   `tfsdk:"nca_id"`
	FunctionName             types.String   `tfsdk:"function_name"`
	HelmChartUri             types.String   `tfsdk:"helm_chart_uri"`
	HelmChartServiceName     types.String   `tfsdk:"helm_chart_service_name"`
	HelmChartServicePort     types.Int64    `tfsdk:"helm_chart_service_port"`
	ContainerImageUri        types.String   `tfsdk:"container_image_uri"`
	ContainerPort            types.Int64    `tfsdk:"container_port"`
	EndpointPath             types.String   `tfsdk:"endpoint_path"`
	HealthEndpointPath       types.String   `tfsdk:"health_endpoint_path"`
	APIBodyFormat            types.String   `tfsdk:"api_body_format"`
	DeploymentSpecifications types.List     `tfsdk:"deployment_specifications"`
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
	data.FunctionName = types.StringValue(functionInfo.Name)
	data.FunctionID = userProvideFunctionID
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
				Required:            true,
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
			"keep_failed_resource": schema.BoolAttribute{
				MarkdownDescription: "Don't delete failed resource. Default is \"false\"",
				Optional:            true,
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

	var createNvidiaCloudFunctionResponse, err = r.client.CreateNvidiaCloudFunction(ctx, data.FunctionID.ValueString(),
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
			"Failed to create Cloud Function",
			"Got unexpected result when creating Cloud Function",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	function := createNvidiaCloudFunctionResponse.Function

	if len(data.DeploymentSpecifications.Elements()) == 0 {
		_, err := r.client.DeleteNvidiaCloudFunctionDeployment(ctx, data.Id.ValueString(), data.VersionID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to delete the old Cloud Function deployment",
				"Got unexpected result when deleting Cloud Function deployment",
			)
		}
		if resp.Diagnostics.HasError() {
			return
		}
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
			"Failed to update Cloud Function",
			"Got unexpected result when updating Cloud Function",
		)
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
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version_id"), idParts[1])...)
}
