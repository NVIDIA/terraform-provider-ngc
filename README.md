# Terraform Provider NGC (Terraform Plugin Framework)

_This repository is built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework). The repository built on the [Terraform Plugin SDK](https://github.com/hashicorp/terraform-plugin-sdk) can be found at [terraform-provider-scaffolding](https://github.com/hashicorp/terraform-provider-scaffolding). See [Which SDK Should I Use?](https://developer.hashicorp.com/terraform/plugin/framework-benefits) in the Terraform documentation for additional information._

This repository is a NGC Terraform Provider which building from the *template* for a [Terraform](https://www.terraform.io) provider, containing:

- Resources and datasources (`internal/provider/`),
- Examples (`examples/`) and generated documentation (`docs/`),
- Miscellaneous meta files.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install .
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

Fill this in for each provider

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```

## Testing the Provider Locally

1. Update the `~/terraformrc`

    ```terraform
    provider_installation {
        dev_overrides {
            "nvidia.com/dev/ngc" = "/home/huaweic/go/bin" # The path of go bin.
        }
        # For all other providers, install them directly from their origin provider
        # registries as normal. If you omit this, Terraform will _only_ use
        # the dev_overrides block, and so no other providers will be available.
        direct {}
    }
    ```

2. Change debug level `export TF_LOG=DEBUG`
3. Setup NGC key `export NGC_API_KEY=nvapi-REDACTED`
4. Test with terraform HCL. Here is an example. Please replace `backend` and `instance_type` by yourselves.

    ```terraform
        terraform {
            required_providers {
                ngc = {
                source = "nvidia.com/dev/ngc"
                }
            }
        }

        provider "ngc" {
            ngc_org  = "shhh2i6mga69" # Omniverse Cloud Prod
            ngc_team = "devinfra"
        }

        resource "ngc_cloud_function" "helm_based_cloud_function_example" {
            function_name           = "terraform-cloud-function-resource-example-helm"
            helm_chart          = "https://helm.ngc.nvidia.com/shhh2i6mga69/devinfra/charts/inference-test-0.1.tgz"
            helm_chart_service_name = "entrypoint"
            inference_port = 8000
            inference_url           = "/echo"
            health_uri    = "/health"
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
        }

        resource "ngc_cloud_function" "helm_based_cloud_function_example_version" {
            function_name           = ngc_cloud_function.helm_based_cloud_function_example.function_name
            function_id             = ngc_cloud_function.helm_based_cloud_function_example.id
            helm_chart          = "https://helm.ngc.nvidia.com/shhh2i6mga69/devinfra/charts/inference-test-0.1.tgz"
            helm_chart_service_name = "entrypoint"
            inference_port = 8000
            inference_url           = "/echo"
            health_uri    = "/health"
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
        }

        resource "ngc_cloud_function" "container_based_cloud_function_example" {
            function_name        = "terraform-cloud-function-resource-example-container"
            container_image  = "nvcr.io/shhh2i6mga69/devinfra/fastapi_echo_sample:latest"
            inference_port       = 8000
            inference_url        = "/echo"
            health_uri = "/health"
            api_body_format      = "CUSTOM"
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
        }

        resource "ngc_cloud_function" "container_based_cloud_function_example_version" {
            function_name        = ngc_cloud_function.container_based_cloud_function_example.function_name
            function_id          = ngc_cloud_function.container_based_cloud_function_example.id
            container_image  = "nvcr.io/shhh2i6mga69/devinfra/fastapi_echo_sample:latest"
            inference_port       = 8000
            inference_url        = "/echo"
            health_uri = "/health"
            api_body_format      = "CUSTOM"
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
        }

        data "ngc_cloud_function" "terraform-cloud-function-datasource-example" {
            function_id = "fe97aa46-c8ea-4237-ba56-1212036f4d0f"
            version_id  = "868d2192-6819-4b53-89f5-3c7fb1df2a72"
        }

        output "function_details" {
            value = data.ngc_cloud_function.terraform-cloud-function-datasource-example
        }
    ```

## Executing Acceptence Test

1. Prepare testconfig file. Default is `./test-config.env`

2. Create NGC personal key with `Function Management` permissions

3. Execute test

```sh
# Run with default test config
make NGC_API_KEY=nvapi-REDACTED testacc

# Run with custom test config
make TEST_ENV_FILE={{ file path }} NGC_API_KEY=nvapi-REDACTED testacc
```

## Executing Unit Test

```sh
make test
```
