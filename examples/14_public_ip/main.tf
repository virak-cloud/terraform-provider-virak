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

# Create a network first
resource "virakcloud_network" "example" {
  zone_id   = var.zone_id
  name      = "public-ip-example-network"
  cidr      = "192.168.1.0/24"
  offering_id = var.network_offering_id
}

# Create an instance
resource "virakcloud_instance" "example" {
  zone_id            = var.zone_id
  name               = "public-ip-example-instance"
  service_offering_id = var.instance_offering_id
  vm_image_id        = var.vm_image_id
  network_ids        = [virakcloud_network.example.id]
}

# Allocate a public IP
resource "virakcloud_public_ip_association" "example" {
  network_id = virakcloud_network.example.id
}

# Allocate a public IP with Static NAT enabled
resource "virakcloud_public_ip" "with_nat" {
  zone_id    = var.zone_id
  network_id = virakcloud_network.example.id
  instance_id = virakcloud_instance.example.id
}