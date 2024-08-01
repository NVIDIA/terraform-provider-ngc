terraform {
  required_providers {
    ngc = {
      source = "nvidia/ngc"
    }
  }
}

provider "ngc" {
  ngc_api_key = "nvapi-REDACTED" # Can be replaced with `NGC_API_KEY` environment variable.
  ngc_org     = "shhh2i6mga69"   # Can be replace with `NGC_ORG` environment variable.
  ngc_team    = "devinfra"       # Can be replace with `NGC_TEAM` environment variable.
}
