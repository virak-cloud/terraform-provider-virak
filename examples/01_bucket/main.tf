# Provider configuration
provider "virakcloud" {
  token = var.virakcloud_token
}

# Data Sources - Fetch available zones
data "virakcloud_zones" "available" {}

# Bucket Resource - Private bucket
resource "virakcloud_bucket" "private_bucket" {
  name    = "my-private-bucket-${random_string.suffix.result}"
  zone_id = data.virakcloud_zones.available.zones[0].id
  policy  = "Private"
}

# Bucket Resource - Public bucket
resource "virakcloud_bucket" "public_bucket" {
  name    = "my-public-bucket-${random_string.suffix.result}"
  zone_id = data.virakcloud_zones.available.zones[0].id
  policy  = "Public"
}

# Random suffix to ensure unique bucket names
resource "random_string" "suffix" {
  length  = 8
  lower   = true
  upper   = false
  numeric = true
  special = false
}