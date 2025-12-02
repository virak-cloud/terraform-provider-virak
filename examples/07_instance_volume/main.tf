# Provider configuration
provider "virakcloud" {
  token = var.virakcloud_token
}

# Data Sources - Fetch available zones
data "virakcloud_zones" "available" {}

# Data Sources - Fetch available instance service offerings
data "virakcloud_instance_service_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Data Sources - Fetch available instance images
data "virakcloud_instance_images" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Data Sources - Fetch available volume service offerings
data "virakcloud_volume_service_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Data Sources - Fetch available networks
data "virakcloud_networks" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Random suffix to ensure unique names
resource "random_string" "suffix" {
  length  = 8
  lower   = true
  upper   = true
  numeric = true
  special = false
}

# Instance Resource - Create an instance with inline volumes
resource "virakcloud_instance" "example" {
  name                = "test-instance-${random_string.suffix.result}"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  service_offering_id = data.virakcloud_instance_service_offerings.available.offerings[0].id
  vm_image_id         = data.virakcloud_instance_images.available.images[0].id
  network_ids         = [data.virakcloud_networks.available.networks[0].id]
}



resource "virakcloud_volume" "example" {
  name                = "test-volume-${random_string.suffix.result}"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  service_offering_id = data.virakcloud_volume_service_offerings.available.offerings[0].id
  size                = 25
  instance_id         = virakcloud_instance.example.id
}