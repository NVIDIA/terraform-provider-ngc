---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "ngc Provider"
subcategory: ""
description: |-
  
---

# ngc Provider



## Example Usage

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `ngc_api_key` (String, Sensitive) NGC Personal Token with `Cloud Function` permission
- `ngc_endpoint` (String) NGC API endpoint
- `ngc_org` (String) NGC Org Name.
- `ngc_team` (String) NGC Team Name
