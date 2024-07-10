---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "ngc_cloud_function Resource - ngc"
subcategory: ""
description: |-
  Nvidia Cloud Function Resource
---

# ngc_cloud_function (Resource)

Nvidia Cloud Function Resource

## Example Usage

```terraform
resource "ngc_cloud_function" "helm_based_cloud_function_example" {
  function_name           = "terraform-cloud-function-resource-example-helm"
  helm_chart_uri          = "https://helm.ngc.nvidia.com/shhh2i6mga69/devinfra/charts/inference-test-0.1.tgz"
  helm_chart_service_name = "entrypoint"
  helm_chart_service_port = 8000
  endpoint_path           = "/echo"
  health_endpoint_path    = "/health"
  api_body_format         = "CUSTOM"
  deployment_specifications = [
    {
      configuration           = "{\"image\":{\"repository\":\"nvcr.io/shhh2i6mga69/devinfra/fastapi_echo_sample\",\"tag\":\"latest\"}}",
      backend                 = "GFN"
      gpu_type                = "L40"
      max_instances           = 1
      min_instances           = 1
      max_request_concurrency = 1
    }
  ]
}

resource "ngc_cloud_function" "helm_based_cloud_function_example_version" {
  function_name           = ngc_cloud_function.helm_based_cloud_function_example.function_name
  function_id             = ngc_cloud_function.helm_based_cloud_function_example.id
  helm_chart_uri          = "https://helm.ngc.nvidia.com/shhh2i6mga69/devinfra/charts/inference-test-0.1.tgz"
  helm_chart_service_name = "entrypoint"
  helm_chart_service_port = 8000
  endpoint_path           = "/echo"
  health_endpoint_path    = "/health"
  api_body_format         = "CUSTOM"
  deployment_specifications = [
    {
      configuration           = "{\"image\":{\"repository\":\"nvcr.io/shhh2i6mga69/devinfra/fastapi_echo_sample\",\"tag\":\"latest\"}}",
      backend                 = "GFN"
      gpu_type                = "L40"
      max_instances           = 1
      min_instances           = 1
      max_request_concurrency = 1
    }
  ]
}

resource "ngc_cloud_function" "container_based_cloud_function_example" {
  function_name        = "terraform-cloud-function-resource-example-container"
  container_image_uri  = "nvcr.io/shhh2i6mga69/devinfra/fastapi_echo_sample:latest"
  container_port       = 8000
  endpoint_path        = "/echo"
  health_endpoint_path = "/health"
  api_body_format      = "CUSTOM"
  deployment_specifications = [
    {
      backend                 = "GFN"
      gpu_type                = "L40"
      max_instances           = 1
      min_instances           = 1
      max_request_concurrency = 1
    }
  ]
}

resource "ngc_cloud_function" "container_based_cloud_function_example_version" {
  function_name        = ngc_cloud_function.container_based_cloud_function_example.function_name
  function_id          = ngc_cloud_function.container_based_cloud_function_example.id
  container_port       = 8000
  endpoint_path        = "/echo"
  health_endpoint_path = "/health"
  api_body_format      = "CUSTOM"
  deployment_specifications = [
    {
      backend                 = "GFN"
      gpu_type                = "L40"
      max_instances           = 1
      min_instances           = 1
      max_request_concurrency = 1
    }
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `function_name` (String) Function name

### Optional

- `api_body_format` (String) API Body Format. Default is "CUSTOM"
- `container_image_uri` (String) Container image uri
- `container_port` (Number) Container port
- `deployment_specifications` (Attributes List) (see [below for nested schema](#nestedatt--deployment_specifications))
- `endpoint_path` (String) Service endpoint Path. Default is "/"
- `function_id` (String) Function ID
- `health_endpoint_path` (String) Service health endpoint Path. Default is "/v2/health/ready"
- `helm_chart_service_name` (String) Target service name
- `helm_chart_service_port` (Number) Target service port
- `helm_chart_uri` (String) Helm chart registry uri

### Read-Only

- `id` (String) Read-only Function ID
- `nca_id` (String) NCA ID
- `version_id` (String) Function Version ID

<a id="nestedatt--deployment_specifications"></a>
### Nested Schema for `deployment_specifications`

Required:

- `backend` (String) NVCF Backend, default is GFN.
- `gpu_type` (String) GPU Type, GFN backend default is L40
- `max_instances` (Number) Max Instances Count
- `max_request_concurrency` (Number) Max Concurrency Count
- `min_instances` (Number) Min Instances Count

Optional:

- `configuration` (String) Will be the json definition to overwrite the existing values.yaml file when deploying Helm-Based Functions