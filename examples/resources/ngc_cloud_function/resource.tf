resource "ngc_cloud_function" "helm_based_cloud_function_example" {
  function_name           = "terraform-cloud-function-resource-example-helm"
  helm_chart              = "https://helm.ngc.nvidia.com/shhh2i6mga69/devinfra/charts/inference-test-0.1.tgz"
  helm_chart_service_name = "entrypoint"
  inference_port          = 8000
  inference_url           = "/echo"
  api_body_format         = "CUSTOM"
  deployment_specifications = [
    {
      configuration           = "{\"image\":{\"repository\":\"nvcr.io/shhh2i6mga69/devinfra/fastapi_echo_sample\",\"tag\":\"latest\"}}",
      backend                 = "dgxc-forge-az33-prd1"
      instance_type           = "DGX-CLOUD.GPU.L40_1x"
      gpu_type                = "L40"
      max_instances           = 1
      min_instances           = 1
      max_request_concurrency = 1
    }
  ]
  health = {
    uri                  = "/health"
    port                 = 8000
    expected_status_code = 200
    timeout              = "PT10S"
    protocol             = "HTTP"
  }
  tags = [
    "test"
  ]
  keep_failed_resource = true
  timeouts = {
    create = "10m"
  }
}

resource "ngc_cloud_function" "helm_based_cloud_function_example_version" {
  function_name           = ngc_cloud_function.helm_based_cloud_function_example.function_name
  function_id             = ngc_cloud_function.helm_based_cloud_function_example.id
  helm_chart              = "https://helm.ngc.nvidia.com/shhh2i6mga69/devinfra/charts/inference-test-0.1.tgz"
  helm_chart_service_name = "entrypoint"
  inference_port          = 8000
  inference_url           = "/echo"
  api_body_format         = "CUSTOM"
  deployment_specifications = [
    {
      configuration           = "{\"image\":{\"repository\":\"nvcr.io/shhh2i6mga69/devinfra/fastapi_echo_sample\",\"tag\":\"latest\"}}",
      backend                 = "dgxc-forge-az33-prd1"
      instance_type           = "DGX-CLOUD.GPU.L40_1x"
      gpu_type                = "L40"
      max_instances           = 1
      min_instances           = 1
      max_request_concurrency = 1
    }
  ]
  health = {
    uri                  = "/health"
    port                 = 8000
    expected_status_code = 200
    timeout              = "PT10S"
    protocol             = "HTTP"
  }
}

resource "ngc_cloud_function" "container_based_cloud_function_example" {
  function_name   = "terraform-cloud-function-resource-example-container"
  container_image = "nvcr.io/shhh2i6mga69/devinfra/fastapi_echo_sample:latest"
  inference_port  = 8000
  inference_url   = "/echo"
  api_body_format = "CUSTOM"
  deployment_specifications = [
    {
      backend                 = "dgxc-forge-az33-prd1"
      instance_type           = "DGX-CLOUD.GPU.L40_1x"
      gpu_type                = "L40"
      max_instances           = 1
      min_instances           = 1
      max_request_concurrency = 1
    }
  ]
  container_environment = [
    {
      key   = "mock1",
      value = "mock2"
    },
    {
      key   = "mock3",
      value = "mock4"
    },
    {
      key   = "mock5",
      value = "mock6"
    }
  ]
  health = {
    uri                  = "/health"
    port                 = 8000
    expected_status_code = 200
    timeout              = "PT10S"
    protocol             = "HTTP"
  }
  tags = [
    "test"
  ]
}

resource "ngc_cloud_function" "container_based_cloud_function_example_version" {
  function_name   = ngc_cloud_function.container_based_cloud_function_example.function_name
  function_id     = ngc_cloud_function.container_based_cloud_function_example.id
  container_image = "nvcr.io/shhh2i6mga69/devinfra/fastapi_echo_sample:latest"
  inference_port  = 8000
  inference_url   = "/echo"
  health_uri      = "/health"
  api_body_format = "CUSTOM"
  models = [
    {
      name    = "gemma_2b_base_2"
      version = "1.1"
      uri     = "/v2/org/nvidia/team/nemo/models/gemma_2b_base/1.1/files"
    },
    {
      name    = "gemma_2b_base_1"
      version = "1.1"
      uri     = "/v2/org/nvidia/team/nemo/models/gemma_2b_base/1.1/files"
    },
    {
      name    = "gemma_2b_base_3"
      version = "1.1"
      uri     = "/v2/org/nvidia/team/nemo/models/gemma_2b_base/1.1/files"
    },
    {
      name    = "gemma_2b_base"
      version = "1.1"
      uri     = "/v2/org/nvidia/team/nemo/models/gemma_2b_base/1.1/files"
    },
  ]
  deployment_specifications = [
    {
      backend                 = "dgxc-forge-az33-prd1"
      instance_type           = "DGX-CLOUD.GPU.L40_1x"
      gpu_type                = "L40"
      max_instances           = 1
      min_instances           = 1
      max_request_concurrency = 1
    }
  ]
  container_environment = [
    {
      key   = "test1"
      value = "test1"
    },
    {
      key   = "test2"
      value = "test2"
    },
    {
      key   = "test3"
      value = "test3"
    }
  ]
  health = {
    uri                  = "/health"
    port                 = 8000
    expected_status_code = 200
    timeout              = "PT10S"
    protocol             = "HTTP"
  }
  secrets = [
    {
      name  = "test-raw"
      value = "test-raw"
    },
    {
      name  = "test-json",
      value = <<EOF
      {
        "AWS_REGION": "us-west-2",
        "AWS_BUCKET": "content",
        "AWS_ACCESS_KEY_ID": "content-key-id",
        "AWS_SECRET_ACCESS_KEY": "content-access-key",
        "AWS_SESSION_TOKEN": "content-session-token"
      }
      EOF
    },
    {
      name  = "test.s3.us-west-2.amazonaws.com",
      value = <<EOF
      {
        "AWS_ACCESS_KEY_ID" : "s3.us-west-2-key-id",
        "AWS_SECRET_ACCESS_KEY" : "s3.us-west-2-access-key"
      }
      EOF
    }
  ]
  tags = [
    "test"
  ]
}
