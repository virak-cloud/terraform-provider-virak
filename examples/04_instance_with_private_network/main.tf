# Provider configuration
provider "virakcloud" {
  token = var.virakcloud_token
}

# Data Sources - Fetch available zones
data "virakcloud_zones" "available" {}

# Data Sources - Fetch available network offerings
data "virakcloud_network_service_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Data Sources - Fetch available instance service offerings
data "virakcloud_instance_service_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Data Sources - Fetch available instance images
data "virakcloud_instance_images" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Private L3 Network Resource
resource "virakcloud_network" "private_network" {
  name                = "test-private-network-${random_string.suffix.result}"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  network_offering_id = data.virakcloud_network_service_offerings.available.offerings[0].id
  type                = "L3"
  gateway             = "192.168.1.1"
  netmask             = "255.255.255.0"
}

# Instance Resource - Create an instance attached to the private network
resource "virakcloud_instance" "private_network_instance" {
  depends_on = [virakcloud_network.private_network]

  name                = "test-instance-private-network-${random_string.suffix.result}"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  service_offering_id = data.virakcloud_instance_service_offerings.available.offerings[0].id
  vm_image_id         = data.virakcloud_instance_images.available.images[0].id
  network_ids         = [virakcloud_network.private_network.id]
}

# Random suffix to ensure unique resource names
resource "random_string" "suffix" {
  length  = 8
  lower   = true
  upper   = false
  numeric = true
  special = false
}