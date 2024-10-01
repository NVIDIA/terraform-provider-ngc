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
	"sync"
)

type NGCClient struct {
	NgcEndpoint string
	NgcApiKey   string
	NgcOrg      string
	NgcTeam     string
	HttpClient  *http.Client
}

var nvcfClient *NVCFClient = nil
var nvcfClientOnce sync.Once

func (c *NGCClient) NVCFClient() *NVCFClient {
	nvcfClientOnce.Do(func() {
		nvcfClient = &NVCFClient{c.NgcEndpoint, c.NgcApiKey, c.NgcOrg, c.NgcTeam, c.HttpClient}
	})
	return nvcfClient
}
