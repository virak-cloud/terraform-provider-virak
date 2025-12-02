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

# Data Sources - Fetch available network service offerings
data "virakcloud_network_service_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Data Sources - Fetch all networks in the zone
data "virakcloud_networks" "all" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Filter for public networks
locals {
  public_networks = [
    for net in data.virakcloud_networks.all.networks : net if net.type == "Shared"
  ]
  public_network_id = length(local.public_networks) > 0 ? local.public_networks[0].id : null
}

resource "virakcloud_network" "private_network" {
  name                = "test-private-network-${random_string.suffix.result}"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  network_offering_id = data.virakcloud_network_service_offerings.available.offerings[0].id
  type                = "L3"
  gateway             = "192.168.1.1"
  netmask             = "255.255.255.0"
}

# Instance Resource - Create an instance attached to the public network
resource "virakcloud_instance" "public_network_instance" {
  name                = "test-instance-public-network-${random_string.suffix.result}"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  service_offering_id = data.virakcloud_instance_service_offerings.available.offerings[0].id
  vm_image_id         = data.virakcloud_instance_images.available.images[0].id
  network_ids         = local.public_network_id != null ? [local.public_network_id, virakcloud_network.private_network.id] : [virakcloud_network.private_network.id]
}

# Random suffix to ensure unique instance names
resource "random_string" "suffix" {
  length  = 8
  lower   = true
  upper   = false
  numeric = true
  special = false
}