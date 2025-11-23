terraform {
  required_providers {
    virakcloud = {
      source = "registry.terraform.io/virak-cloud/virak-cloud"
      version = ">= 0.1"
    }
  }
}

provider "virakcloud" {
  token    = var.virakcloud_token
}

# Data sources for dynamic resource selection
data "virakcloud_zones" "available" {}
data "virakcloud_instance_offerings" "available" {}
data "virakcloud_instance_images" "available" {}
data "virakcloud_volume_offerings" "available" {}

# Network for instances
resource "virakcloud_network" "volume_example_network" {
  name         = "advanced-volumes-network"
  cidr         = "192.168.30.0/24"
  gateway      = "192.168.30.1"
  netmask      = "255.255.255.0"
  network_type = "L3"
  zone_id      = data.virakcloud_zones.available.zone_id
}

# Base instance to attach volumes to
resource "virakcloud_instance" "volume_host" {
  name = "volume-host-instance"

  offering_id = data.virakcloud_instance_offerings.available.offering_id
  image_id    = data.virakcloud_instance_images.available.image_id
  zone_id     = data.virakcloud_zones.available.zone_id
}

# Root volume for OS
resource "virakcloud_volume" "root_volume" {
  name       = "root-volume"
  size       = 50  # GB - larger for OS and applications
  offering_id = data.virakcloud_volume_offerings.available.offering_id
  zone_id     = data.virakcloud_zones.available.zone_id
}

# Data volume for databases and persistent storage
resource "virakcloud_volume" "data_volume" {
  name       = "data-volume"
  size       = 100  # GB - larger for data storage
  offering_id = data.virakcloud_volume_offerings.available.offering_id
  zone_id     = data.virakcloud_zones.available.zone_id
}

# Backup volume for archives and snapshots
resource "virakcloud_volume" "backup_volume" {
  name       = "backup-volume"
  size       = 200  # GB - largest for backups
  offering_id = data.virakcloud_volume_offerings.available.offering_id
  zone_id     = data.virakcloud_zones.available.zone_id
}

# Standalone volumes (for future use or migration)
resource "virakcloud_volume" "staging_volume" {
  name       = "staging-volume"
  size       = 20   # GB - smaller for staging
  offering_id = data.virakcloud_volume_offerings.available.offering_id
  zone_id     = data.virakcloud_zones.available.zone_id
}

# Migration target instance (for volume migration concepts)
resource "virakcloud_instance" "migration_target" {
  name                = "migration-target-instance"
  zone_id             = data.virakcloud_zones.available.zone_id
  service_offering_id = data.virakcloud_instance_service_offerings.available.offering_id
  vm_image_id         = data.virakcloud_instance_images.available.image_id
  network_ids         = [virakcloud_network.volume_example_network.id]

  depends_on = [virakcloud_instance.volume_host]
}

# Example of lifecycle management with volumes
resource "virakcloud_volume" "temporary_volume" {
  name       = "temporary-volume"
  size       = 10   # GB - temp volume
  offering_id = data.virakcloud_volume_offerings.available.offering_id
  zone_id     = data.virakcloud_zones.available.zone_id
}
