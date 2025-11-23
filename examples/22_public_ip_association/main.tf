terraform {
  required_providers {
    virakcloud = {
      source = "virak-cloud/virakcloud"
    }
  }
}

provider "virakcloud" {
  # token   = "" # Set via VIRAKCLOUD_TOKEN environment variable
}

resource "virakcloud_network" "example" {
  zone_id     = var.zone_id
  name        = "public-ip-assoc-example-network"
  cidr        = "192.168.1.0/24"
  offering_id = var.network_offering_id
}

resource "virakcloud_public_ip_association" "example" {
  network_id = virakcloud_network.example.id
}
