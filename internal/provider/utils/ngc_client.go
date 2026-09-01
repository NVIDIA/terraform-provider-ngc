//  SPDX-FileCopyrightText: Copyright (c) 2024 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
//  SPDX-License-Identifier: LicenseRef-NvidiaProprietary

//  NVIDIA CORPORATION, its affiliates and licensors retain all intellectual
//  property and proprietary rights in and to this material, related
//  documentation and any modifications thereto. Any use, reproduction,
//  disclosure or distribution of this material and related documentation
//  without an express license agreement from NVIDIA CORPORATION or
//  its affiliates is strictly prohibited.

package utils

import (
	"net/http"
)

type NGCClient struct {
	NgcEndpoint string
	NgcApiKey   string
	NgcOrg      string
	NgcTeam     string
	HttpClient  *http.Client
}

// NVCFClient returns an NVCF client bound to this NGCClient's credentials.
//
// This deliberately does not cache in a package-level singleton. Terraform can
// instantiate several aliased `ngc` providers in one plugin process, each with
// its own ngc_api_key, ngc_org and ngc_endpoint; a process-wide singleton would
// serve every alias the credentials of whichever one happened to configure
// first, sending one org's key to another org's endpoint. NVCFClient is a small
// immutable value and shares the caller's *http.Client, so per-instance
// construction keeps connection pooling and adds no meaningful cost.
func (c *NGCClient) NVCFClient() *NVCFClient {
	return &NVCFClient{c.NgcEndpoint, c.NgcApiKey, c.NgcOrg, c.NgcTeam, c.HttpClient}
}
