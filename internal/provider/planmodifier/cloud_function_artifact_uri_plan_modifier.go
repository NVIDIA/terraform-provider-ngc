package custom_planmodifier

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CloudFunctionArtifactUriPlanModifier struct{}

func (m CloudFunctionArtifactUriPlanModifier) Description(ctx context.Context) string {
	return "Automatically adds artifact host name to URI if missing"
}

func (m CloudFunctionArtifactUriPlanModifier) MarkdownDescription(ctx context.Context) string {
	return "Automatically adds artifact host name to URI if missing"
}

func (m CloudFunctionArtifactUriPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	value := req.PlanValue.ValueString()
	if value == "" {
		value = req.StateValue.ValueString()
	}

	if value == "" {
		return
	}

	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return
	}

	defaultHost := os.Getenv("NGC_ENDPOINT")

	if defaultHost == "" {
		defaultHost = "https://api.ngc.nvidia.com"
	}
	resp.PlanValue = types.StringValue(fmt.Sprintf("%s/%s", defaultHost, strings.TrimPrefix(value, "/")))
}
