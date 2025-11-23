# Provider configuration
provider "virakcloud" {
  token    = var.virakcloud_token
}

# Data Sources
data "virakcloud_zones" "available" {}
data "virakcloud_network_service_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
  type    = "Isolated"
}

# Network Resource
resource "virakcloud_network" "vpn_network" {
  name                = "example-vpn-network"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  network_offering_id = data.virakcloud_network_service_offerings.available.offerings[0].id
  type                = "Isolated"
  gateway             = "192.168.1.1"
  netmask             = "255.255.255.0"
}

# VPN Resource
resource "virakcloud_network_vpn" "example" {
  zone_id     = data.virakcloud_zones.available.zones[0].id
  network_id  = virakcloud_network.vpn_network.id
  enabled     = true
  username    = "vpnuser"
  password    = "securepassword123"
}