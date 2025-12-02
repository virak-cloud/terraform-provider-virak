provider "virakcloud" {
  token = var.virakcloud_token
}

data "virakcloud_zones" "available" {}

data "virakcloud_instance_service_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

data "virakcloud_instance_images" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

data "virakcloud_volume_service_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

data "virakcloud_network_service_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

locals {
  primary_zone_id      = data.virakcloud_zones.available.zones[0].id
  instance_offering_id = data.virakcloud_instance_service_offerings.available.offerings[0].id
  volume_offering_id   = data.virakcloud_volume_service_offerings.available.offerings[0].id
  network_offering_id  = data.virakcloud_network_service_offerings.available.offerings[0].id
}

resource "random_string" "suffix" {
  length  = 6
  lower   = true
  upper   = false
  numeric = true
  special = false
}

resource "virakcloud_network" "volume_example_network" {
  name                = "advanced-volumes-network-${random_string.suffix.result}"
  zone_id             = local.primary_zone_id
  network_offering_id = local.network_offering_id
  type                = "L3"
  gateway             = "192.168.30.1"
  netmask             = "255.255.255.0"
}

resource "virakcloud_instance" "volume_host" {
  name                = "volume-host-${random_string.suffix.result}"
  zone_id             = local.primary_zone_id
  service_offering_id = local.instance_offering_id
  vm_image_id         = data.virakcloud_instance_images.available.images[0].id
  network_ids         = [virakcloud_network.volume_example_network.id]
}

resource "virakcloud_instance" "migration_target" {
  name                = "migration-target-${random_string.suffix.result}"
  zone_id             = local.primary_zone_id
  service_offering_id = local.instance_offering_id
  vm_image_id         = data.virakcloud_instance_images.available.images[0].id
  network_ids         = [virakcloud_network.volume_example_network.id]

  depends_on = [virakcloud_instance.volume_host]
}

resource "virakcloud_volume" "root_volume" {
  name                = "root-volume-${random_string.suffix.result}"
  zone_id             = local.primary_zone_id
  service_offering_id = local.volume_offering_id
  size                = 50
  instance_id         = virakcloud_instance.volume_host.id
}

resource "virakcloud_volume" "data_volume" {
  name                = "data-volume-${random_string.suffix.result}"
  zone_id             = local.primary_zone_id
  service_offering_id = local.volume_offering_id
  size                = 100
  instance_id         = virakcloud_instance.volume_host.id
}

resource "virakcloud_volume" "backup_volume" {
  name                = "backup-volume-${random_string.suffix.result}"
  zone_id             = local.primary_zone_id
  service_offering_id = local.volume_offering_id
  size                = 200
}

resource "virakcloud_volume" "staging_volume" {
  name                = "staging-volume-${random_string.suffix.result}"
  zone_id             = local.primary_zone_id
  service_offering_id = local.volume_offering_id
  size                = 20
}

resource "virakcloud_volume" "temporary_volume" {
  name                = "temporary-volume-${random_string.suffix.result}"
  zone_id             = local.primary_zone_id
  service_offering_id = local.volume_offering_id
  size                = 10
  instance_id         = virakcloud_instance.volume_host.id
}
