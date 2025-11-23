provider "virakcloud" {
  token    = var.virakcloud_token
}

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

data "virakcloud_networks" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

locals {
  public_networks = [
    for net in data.virakcloud_networks.available.networks : net if net.network_offering.type == "Shared"
  ]
  public_network_id = length(local.public_networks) > 0 ? local.public_networks[0].id : null
}

resource "virakcloud_instance" "example" {
  name                = "lifecycle-example-instance"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  service_offering_id = data.virakcloud_instance_offerings.available.offerings[0].id
  vm_image_id         = data.virakcloud_instance_images.available.images[0].id
  network_ids         = local.public_network_id != null ? [local.public_network_id] : []

  desired_state = "running"
}


