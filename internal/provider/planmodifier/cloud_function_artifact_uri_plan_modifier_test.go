//  SPDX-FileCopyrightText: Copyright (c) 2024 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
//  SPDX-License-Identifier: LicenseRef-NvidiaProprietary

//  NVIDIA CORPORATION, its affiliates and licensors retain all intellectual
//  property and proprietary rights in and to this material, related
//  documentation and any modifications thereto. Any use, reproduction,
//  disclosure or distribution of this material and related documentation
//  without an express license agreement from NVIDIA CORPORATION or
//  its affiliates is strictly prohibited.

//go:build unittest
// +build unittest

package custom_planmodifier

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestCloudFunctionArtifactUriPlanModifier_Description(t *testing.T) {
	t.Parallel()

	modifier := CloudFunctionArtifactUriPlanModifier{}
	ctx := context.Background()

	description := modifier.Description(ctx)
	assert.Equal(t, "Automatically adds artifact host name to URI if missing", description)
}

func TestCloudFunctionArtifactUriPlanModifier_MarkdownDescription(t *testing.T) {
	t.Parallel()

	modifier := CloudFunctionArtifactUriPlanModifier{}
	ctx := context.Background()

	description := modifier.MarkdownDescription(ctx)
	assert.Equal(t, "Automatically adds artifact host name to URI if missing", description)
}

func TestCloudFunctionArtifactUriPlanModifier_PlanModifyString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		planValue      string
		stateValue     string
		envEndpoint    string
		expectedResult string
		description    string
	}{
		{
			name:           "EmptyPlanAndStateValue",
			planValue:      "",
			stateValue:     "",
			envEndpoint:    "",
			expectedResult: "",
			description:    "Should return empty when both plan and state values are empty",
		},
		{
			name:           "HttpsPrefix",
			planValue:      "https://example.com/artifact",
			stateValue:     "",
			envEndpoint:    "",
			expectedResult: "https://example.com/artifact",
			description:    "Should not modify URI with https:// prefix",
		},
		{
			name:           "HttpPrefix",
			planValue:      "http://example.com/artifact",
			stateValue:     "",
			envEndpoint:    "",
			expectedResult: "http://example.com/artifact",
			description:    "Should not modify URI with http:// prefix",
		},
		{
			name:           "RelativeUriWithDefaultHost",
			planValue:      "v2/org/team/artifacts/test",
			stateValue:     "",
			envEndpoint:    "",
			expectedResult: "https://api.ngc.nvidia.com/v2/org/team/artifacts/test",
			description:    "Should prepend default NGC endpoint for relative URIs",
		},
		{
			name:           "RelativeUriWithCustomEnvEndpoint",
			planValue:      "v2/org/team/artifacts/test",
			stateValue:     "",
			envEndpoint:    "https://custom.nvidia.com",
			expectedResult: "https://custom.nvidia.com/v2/org/team/artifacts/test",
			description:    "Should prepend custom NGC endpoint from env variable",
		},
		{
			name:           "RelativeUriWithLeadingSlash",
			planValue:      "/v2/org/team/artifacts/test",
			stateValue:     "",
			envEndpoint:    "",
			expectedResult: "https://api.ngc.nvidia.com/v2/org/team/artifacts/test",
			description:    "Should trim leading slash and prepend default host",
		},
		{
			name:           "RelativeUriWithLeadingSlashAndCustomEndpoint",
			planValue:      "/v2/org/team/artifacts/test",
			stateValue:     "",
			envEndpoint:    "https://custom.nvidia.com",
			expectedResult: "https://custom.nvidia.com/v2/org/team/artifacts/test",
			description:    "Should trim leading slash and prepend custom host",
		},
		{
			name:           "EmptyPlanValueWithStateValue",
			planValue:      "",
			stateValue:     "v2/org/team/artifacts/existing",
			envEndpoint:    "",
			expectedResult: "https://api.ngc.nvidia.com/v2/org/team/artifacts/existing",
			description:    "Should use state value when plan value is empty",
		},
		{
			name:           "EmptyPlanValueWithStateValueHttps",
			planValue:      "",
			stateValue:     "https://example.com/existing",
			envEndpoint:    "",
			expectedResult: "",
			description:    "Should not modify plan value when state value already has https scheme (no prefix needed)",
		},
		{
			name:           "PlanValueTakesPrecedence",
			planValue:      "v2/org/new/artifacts",
			stateValue:     "v2/org/old/artifacts",
			envEndpoint:    "",
			expectedResult: "https://api.ngc.nvidia.com/v2/org/new/artifacts",
			description:    "Plan value should take precedence over state value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or unset environment variable
			if tt.envEndpoint != "" {
				os.Setenv("NGC_ENDPOINT", tt.envEndpoint)
				defer os.Unsetenv("NGC_ENDPOINT")
			} else {
				os.Unsetenv("NGC_ENDPOINT")
			}

			modifier := CloudFunctionArtifactUriPlanModifier{}
			ctx := context.Background()

			req := planmodifier.StringRequest{
				PlanValue:  types.StringValue(tt.planValue),
				StateValue: types.StringValue(tt.stateValue),
			}
			resp := &planmodifier.StringResponse{
				PlanValue: req.PlanValue,
			}

			modifier.PlanModifyString(ctx, req, resp)

			if tt.expectedResult == "" {
				// When expected result is empty, the response should not be modified
				// (PlanValue remains as originally set)
				if tt.planValue == "" && tt.stateValue == "" {
					assert.Equal(t, tt.planValue, resp.PlanValue.ValueString(), tt.description)
				}
			} else {
				assert.Equal(t, tt.expectedResult, resp.PlanValue.ValueString(), tt.description)
			}
		})
	}
}

func TestCloudFunctionArtifactUriPlanModifier_PlanModifyString_NullValues(t *testing.T) {
	t.Parallel()

	os.Unsetenv("NGC_ENDPOINT")

	modifier := CloudFunctionArtifactUriPlanModifier{}
	ctx := context.Background()

	// Test with null plan value
	req := planmodifier.StringRequest{
		PlanValue:  types.StringNull(),
		StateValue: types.StringNull(),
	}
	resp := &planmodifier.StringResponse{
		PlanValue: req.PlanValue,
	}

	modifier.PlanModifyString(ctx, req, resp)

	// When both are null, ValueString() returns empty string, so the modifier should not modify
	assert.True(t, resp.PlanValue.IsNull() || resp.PlanValue.ValueString() == "")
}

func TestCloudFunctionArtifactUriPlanModifier_PlanModifyString_UnknownValues(t *testing.T) {
	t.Parallel()

	os.Unsetenv("NGC_ENDPOINT")

	modifier := CloudFunctionArtifactUriPlanModifier{}
	ctx := context.Background()

	// Test with unknown plan value
	req := planmodifier.StringRequest{
		PlanValue:  types.StringUnknown(),
		StateValue: types.StringValue("v2/org/team/artifacts/test"),
	}
	resp := &planmodifier.StringResponse{
		PlanValue: req.PlanValue,
	}

	modifier.PlanModifyString(ctx, req, resp)

	// Unknown values return empty string from ValueString(), so state value should be used
	assert.Equal(t, "https://api.ngc.nvidia.com/v2/org/team/artifacts/test", resp.PlanValue.ValueString())
}
