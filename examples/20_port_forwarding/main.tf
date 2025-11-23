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

resource "virakcloud_network" "private" {
  name                 = "port-forward-network"
  zone_id              = data.virakcloud_zones.available.zones[0].id
  network_offering_id  = data.virakcloud_network_offerings.available.offerings[0].id
  type                 = "L3"
  gateway              = "192.168.1.1"
  netmask              = "255.255.255.0"
}

resource "virakcloud_public_ip_association" "example" {
  network_id = virakcloud_network.private.id
}

resource "virakcloud_instance" "example" {
  name                = "port-forward-instance"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  service_offering_id = data.virakcloud_instance_offerings.available.offerings[0].id
  vm_image_id         = data.virakcloud_instance_images.available.images[0].id
  network_ids         = [virakcloud_network.private.id]
}

resource "virakcloud_port_forwarding_rule" "example" {
  zone_id      = data.virakcloud_zones.available.zones[0].id
  network_id   = virakcloud_network.private.id
  public_ip_id = virakcloud_public_ip_association.example.id
  protocol     = "TCP"
  public_port  = 8080
  private_port = 80
  instance_id  = virakcloud_instance.example.id
}

