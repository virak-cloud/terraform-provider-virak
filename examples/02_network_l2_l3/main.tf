# Provider configuration
provider "virakcloud" {
  token    = var.virakcloud_token
}

# Data Sources
data "virakcloud_zones" "available" {}
data "virakcloud_network_service_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
  type    = "L2"
}

# L2 Network Resource
resource "virakcloud_network" "l2_network" {
  name                = "example-l2-network-new"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  network_offering_id = data.virakcloud_network_service_offerings.available.offerings[0].id
  type                = "L2"
}

# L3 Network Resource
resource "virakcloud_network" "l3_network" {
  name                = "example-l3-network-2"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  network_offering_id = data.virakcloud_network_service_offerings.available.offerings[0].id
  type                = "Isolated"
  gateway             = "192.168.1.1"
  netmask             = "255.255.255.0"
}