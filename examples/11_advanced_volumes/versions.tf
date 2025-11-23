terraform {
  required_version = ">= 1.0"
  required_providers {
    virakcloud = {
      source  = "registry.terraform.io/virak-cloud/virak-cloud"
      version = ">= 0.1"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.0"
    }
  }
}
