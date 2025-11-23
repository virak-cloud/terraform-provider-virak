terraform {
  required_providers {
    virakcloud = {
      source  = "terraform.local/local/virakcloud"
      version = "~> 0.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}