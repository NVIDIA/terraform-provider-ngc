package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type NVCFClient struct {
	NgcEndpoint string
	NgcApiKey   string
	NgcOrg      string
	NgcTeam     string
	HttpClient  *http.Client
}

func (c *NVCFClient) NvcfEndpoint(context.Context) string {
	if c.NgcTeam == "" {
		return fmt.Sprintf("%s/v2/orgs/%s", c.NgcEndpoint, c.NgcOrg)
	} else {
		return fmt.Sprintf("%s/v2/orgs/%s/teams/%s", c.NgcEndpoint, c.NgcOrg, c.NgcTeam)
	}
}

func (c *NVCFClient) HTTPClient(context.Context) *http.Client {
	return c.HttpClient
}

type ErrorResponse struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

func (c *NVCFClient) sendRequest(ctx context.Context, requestURL string, method string, requestBody any, responseObject any, expectedStatusCode map[int]bool) error {
	var request *http.Request

	if requestBody != nil {
		payloadBuf := new(bytes.Buffer)
		json.NewEncoder(payloadBuf).Encode(requestBody)
		request, _ = http.NewRequest(method, requestURL, payloadBuf)
	} else {
		request, _ = http.NewRequest(method, requestURL, http.NoBody)
	}

	request.Header.Set("Authorization", "Bearer "+c.NgcApiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := c.HttpClient.Do(request)

	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("failed to send request to %s with method %s", requestURL, method))
		return err
	}

	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)

	ctx = tflog.SetField(ctx, "response_status", response.Status)
	ctx = tflog.SetField(ctx, "response_header", response.Header)

	if _, ok := expectedStatusCode[response.StatusCode]; !ok {
		tflog.Error(ctx, "got unexpected response code")

		var errResponseObject = &ErrorResponse{}
		err = json.Unmarshal(body, errResponseObject)

		if err != nil {
			ctx = tflog.SetField(ctx, "response_body", string(body))
			tflog.Error(ctx, "failed to parse error response body")
			return err
		}
		return errors.New(errResponseObject.Detail)
	}

	if responseObject != nil {
		err = json.Unmarshal(body, responseObject)

		if err != nil {
			tflog.Error(ctx, "failed to parse response body")
			return err
		}
	}

	return err
}

type NvidiaCloudFunctionInfo struct {
	ID              string        `json:"id"`
	NcaID           string        `json:"ncaId"`
	VersionID       string        `json:"versionId"`
	Name            string        `json:"name"`
	Status          string        `json:"status"`
	InferenceURL    string        `json:"inferenceUrl"`
	InferencePort   int           `json:"inferencePort"`
	ContainerImage  string        `json:"containerImage"`
	APIBodyFormat   string        `json:"apiBodyFormat"`
	HelmChart       string        `json:"helmChart"`
	HelmServiceName string        `json:"helmChartServiceName"`
	HealthURI       string        `json:"healthUri"`
	CreatedAt       time.Time     `json:"createdAt"`
	Description     string        `json:"description"`
	Health          interface{}   `json:"health"`
	ActiveInstances []interface{} `json:"activeInstances"`
}

type CreateNvidiaCloudFunctionRequest struct {
	FunctionName                       string
	HelmChartUri                       string
	HelmChartValuesOverwriteJsonString string
	HelmChartServiceName               string
	HelmChartServicePort               int
	ContainerImageUri                  string
	ContainerPort                      int
	EndpointPath                       string
	HealthEndpointPath                 string
	APIBodyFormat                      string
}

type CreateNvidiaCloudFunctionResponse struct {
	Function NvidiaCloudFunctionInfo `json:"function"`
}

func (c *NVCFClient) CreateNvidiaCloudFunction(ctx context.Context, functionID string, req CreateNvidiaCloudFunctionRequest) (resp *CreateNvidiaCloudFunctionResponse, err error) {
	if req.ContainerImageUri != "" {
		return c.createContainerBasedNvidiaCloudFunction(ctx, functionID, createContainerBasedNvidiaCloudFunctionRequest{
			FunctionName:       req.FunctionName,
			ContainerPort:      req.ContainerPort,
			ContainerImage:     req.ContainerImageUri,
			APIBodyFormat:      req.APIBodyFormat,
			EndpointPath:       req.EndpointPath,
			HealthEndpointPath: req.HealthEndpointPath,
		})
	} else {
		return c.createHelmBasedNvidiaCloudFunction(ctx, functionID, createHelmBasedNvidiaCloudFunctionRequest{
			FunctionName:         req.FunctionName,
			HelmChartUri:         req.HelmChartUri,
			APIBodyFormat:        req.APIBodyFormat,
			HelmChartServicePort: req.HelmChartServicePort,
			HelmChartServiceName: req.HelmChartServiceName,
			EndpointPath:         req.EndpointPath,
			HealthEndpointPath:   req.HealthEndpointPath,
		})
	}
}

type createHelmBasedNvidiaCloudFunctionRequest struct {
	FunctionName         string `json:"name"`
	HelmChartUri         string `json:"helmChart"`
	HelmChartServiceName string `json:"helmChartServiceName"`
	HelmChartServicePort int    `json:"inferencePort"`
	EndpointPath         string `json:"inferenceUrl"`
	HealthEndpointPath   string `json:"healthUri"`
	APIBodyFormat        string `json:"apiBodyFormat"`
}

func (c *NVCFClient) createHelmBasedNvidiaCloudFunction(ctx context.Context, functionID string, req createHelmBasedNvidiaCloudFunctionRequest) (resp *CreateNvidiaCloudFunctionResponse, err error) {
	var createNvidiaCloudFunctionResponse CreateNvidiaCloudFunctionResponse

	var requestURL string
	if functionID != "" {
		requestURL = fmt.Sprintf("%s/nvcf/functions/%s/versions", c.NvcfEndpoint(ctx), functionID)
	} else {
		requestURL = fmt.Sprintf("%s/nvcf/functions", c.NvcfEndpoint(ctx))
	}

	err = c.sendRequest(ctx, requestURL, http.MethodPost, req, &createNvidiaCloudFunctionResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Create Helm-Based NVCF Function.")
	return &createNvidiaCloudFunctionResponse, err
}

type createContainerBasedNvidiaCloudFunctionRequest struct {
	FunctionName       string `json:"name"`
	ContainerPort      int    `json:"inferencePort"`
	ContainerImage     string `json:"containerImage"`
	EndpointPath       string `json:"inferenceUrl"`
	APIBodyFormat      string `json:"apiBodyFormat"`
	HealthEndpointPath string `json:"healthUri"`
}

func (c *NVCFClient) createContainerBasedNvidiaCloudFunction(ctx context.Context, functionID string, req createContainerBasedNvidiaCloudFunctionRequest) (resp *CreateNvidiaCloudFunctionResponse, err error) {
	var createNvidiaCloudFunctionResponse CreateNvidiaCloudFunctionResponse

	var requestURL string
	if functionID != "" {
		requestURL = fmt.Sprintf("%s/nvcf/functions/%s/versions", c.NvcfEndpoint(ctx), functionID)
	} else {
		requestURL = fmt.Sprintf("%s/nvcf/functions", c.NvcfEndpoint(ctx))
	}

	err = c.sendRequest(ctx, requestURL, http.MethodPost, req, &createNvidiaCloudFunctionResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Create Container-Based NVCF Function.")
	return &createNvidiaCloudFunctionResponse, err
}

type ListNvidiaCloudFunctionVersionsResponse struct {
	Functions []NvidiaCloudFunctionInfo `json:"functions"`
}

type ListNvidiaCloudFunctionVersionsRequest struct {
	FunctionID string `json:"name"`
}

func (c *NVCFClient) ListNvidiaCloudFunctionVersions(ctx context.Context, functionID string) (resp *ListNvidiaCloudFunctionVersionsResponse, err error) {
	var listNvidiaCloudFunctionVersionsResponse ListNvidiaCloudFunctionVersionsResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/functions/" + functionID + "/versions"

	err = c.sendRequest(ctx, requestURL, http.MethodGet, nil, &listNvidiaCloudFunctionVersionsResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "List NVCF Function versions")
	return &listNvidiaCloudFunctionVersionsResponse, err
}

func (c *NVCFClient) DeleteNvidiaCloudFunctionVersion(ctx context.Context, functionID string, functionVersionID string) (err error) {
	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodDelete, nil, nil, map[int]bool{204: true})
	tflog.Debug(ctx, "Delete Function Deployment")
	return err
}

type NvidiaCloudFunctionDeploymentSpecification struct {
	Gpu                   string      `json:"gpu"`
	Backend               string      `json:"backend"`
	InstanceType          string      `json:"instanceType"`
	MaxInstances          int         `json:"maxInstances"`
	MinInstances          int         `json:"minInstances"`
	MaxRequestConcurrency int         `json:"maxRequestConcurrency"`
	Configuration         interface{} `json:"configuration"`
}

type NvidiaCloudFunctionDeployment struct {
	FunctionID               string                                       `json:"functionId"`
	FunctionVersionID        string                                       `json:"functionVersionId"`
	NcaID                    string                                       `json:"ncaId"`
	FunctionStatus           string                                       `json:"functionStatus"`
	HealthInfo               interface{}                                  `json:"healthInfo"`
	DeploymentSpecifications []NvidiaCloudFunctionDeploymentSpecification `json:"deploymentSpecifications"`
}

type CreateNvidiaCloudFunctionDeploymentRequest struct {
	DeploymentSpecifications []NvidiaCloudFunctionDeploymentSpecification `json:"deploymentSpecifications"`
}

type CreateNvidiaCloudFunctionDeploymentResponse struct {
	Deployment NvidiaCloudFunctionDeployment `json:"deployment"`
}

func (c *NVCFClient) CreateNvidiaCloudFunctionDeployment(ctx context.Context, functionID string, functionVersionID string, req CreateNvidiaCloudFunctionDeploymentRequest) (resp *CreateNvidiaCloudFunctionDeploymentResponse, err error) {
	var createNvidiaCloudFunctionDeploymentResponse CreateNvidiaCloudFunctionDeploymentResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/deployments/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodPost, req, &createNvidiaCloudFunctionDeploymentResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Create Function Deployment")
	return &createNvidiaCloudFunctionDeploymentResponse, err
}

type UpdateNvidiaCloudFunctionDeploymentRequest struct {
	DeploymentSpecifications []NvidiaCloudFunctionDeploymentSpecification `json:"deploymentSpecifications"`
}

type UpdateNvidiaCloudFunctionDeploymentResponse struct {
	Deployment NvidiaCloudFunctionDeployment `json:"deployment"`
}

func (c *NVCFClient) UpdateNvidiaCloudFunctionDeployment(ctx context.Context, functionID string, functionVersionID string, req UpdateNvidiaCloudFunctionDeploymentRequest) (resp *UpdateNvidiaCloudFunctionDeploymentResponse, err error) {
	var updateNvidiaCloudFunctionDeploymentResponse UpdateNvidiaCloudFunctionDeploymentResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/deployments/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodPut, req, &updateNvidiaCloudFunctionDeploymentResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Update Function Deployment")
	return &updateNvidiaCloudFunctionDeploymentResponse, err
}

func (c *NVCFClient) WaitingDeploymentCompleted(ctx context.Context, functionID string, functionVersionId string) error {

	for {
		readNvidiaCloudFunctionDeploymentResponse, err := c.ReadNvidiaCloudFunctionDeployment(ctx, functionID, functionVersionId)

		if err != nil {
			return err
		}

		if readNvidiaCloudFunctionDeploymentResponse.Deployment.FunctionStatus == "ACTIVE" {
			return nil
		} else if readNvidiaCloudFunctionDeploymentResponse.Deployment.FunctionStatus == "DEPLOYING" {
			select {
			case <-ctx.Done():
				return errors.New("cancelled")
			case <-time.After(10 * time.Second):
				continue
			}
		} else {
			return fmt.Errorf("unexpected status %s", readNvidiaCloudFunctionDeploymentResponse.Deployment.FunctionStatus)
		}
	}
}

type ReadNvidiaCloudFunctionDeploymentResponse struct {
	Deployment NvidiaCloudFunctionDeployment `json:"deployment"`
}

func (c *NVCFClient) ReadNvidiaCloudFunctionDeployment(ctx context.Context, functionID string, functionVersionID string) (resp *ReadNvidiaCloudFunctionDeploymentResponse, err error) {
	var readNvidiaCloudFunctionDeploymentResponse ReadNvidiaCloudFunctionDeploymentResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/deployments/functions/" + functionID + "/versions/" + functionVersionID

	err = c.sendRequest(ctx, requestURL, http.MethodGet, nil, &readNvidiaCloudFunctionDeploymentResponse, map[int]bool{200: true, 404: true})
	tflog.Debug(ctx, "Read Function Deployment")
	return &readNvidiaCloudFunctionDeploymentResponse, err
}

type DeleteNvidiaCloudFunctionDeploymentResponse struct {
	Function NvidiaCloudFunctionInfo `json:"function"`
}

func (c *NVCFClient) DeleteNvidiaCloudFunctionDeployment(ctx context.Context, functionID string, functionVersionID string) (resp *DeleteNvidiaCloudFunctionDeploymentResponse, err error) {
	var deleteNvidiaCloudFunctionDeploymentResponse DeleteNvidiaCloudFunctionDeploymentResponse

	requestURL := c.NvcfEndpoint(ctx) + "/nvcf/deployments/functions/" + functionID + "/versions/" + functionVersionID
	err = c.sendRequest(ctx, requestURL, http.MethodDelete, nil, &deleteNvidiaCloudFunctionDeploymentResponse, map[int]bool{200: true})
	tflog.Debug(ctx, "Delete Function Deployment")
	return &deleteNvidiaCloudFunctionDeploymentResponse, err
}
