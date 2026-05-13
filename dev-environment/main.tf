# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    youtrack = {
      source = "elcait/youtrack"
    }
  }
}

provider "youtrack" {
  base_url = "http://localhost:8080"
}
