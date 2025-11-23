# Provider configuration
provider "virakcloud" {
  token    = var.virakcloud_token
}

# Data Sources - Fetch available zones and offerings
data "virakcloud_zones" "available" {}
data "virakcloud_instance_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}
data "virakcloud_instance_images" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}
data "virakcloud_network_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Network for the instance
resource "virakcloud_network" "example" {
  zone_id               = data.virakcloud_zones.available.zones[0].id
  network_offering_id   = data.virakcloud_network_offerings.available.network_offerings[0].id
  name                  = "snapshot-example-network"
}

# Instance to snapshot
resource "virakcloud_instance" "example" {
  zone_id             = data.virakcloud_zones.available.zones[0].id
  name                = "snapshot-example-instance"
  service_offering_id = data.virakcloud_instance_offerings.available.instance_offerings[0].id
  vm_image_id         = data.virakcloud_instance_images.available.instance_images[0].id
  network_ids         = [virakcloud_network.example.id]
}

# Snapshot resource
resource "virakcloud_snapshot" "example" {
  zone_id     = data.virakcloud_zones.available.zones[0].id
  instance_id = virakcloud_instance.example.id
  name        = "backup-snapshot"
}