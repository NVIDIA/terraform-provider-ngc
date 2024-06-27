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
