provider "virakcloud" {
  token = var.virakcloud_token
}

data "virakcloud_zones" "available" {}

data "virakcloud_instance_service_offerings" "k8s" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

data "virakcloud_instance_images" "k8s" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

data "virakcloud_network_service_offerings" "isolated" {
  zone_id = data.virakcloud_zones.available.zones[0].id
  type    = "Isolated"
}

data "virakcloud_kubernetes_versions" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

resource "random_string" "suffix" {
  length  = 6
  lower   = true
  upper   = false
  numeric = true
  special = false
}

resource "virakcloud_network" "k8s_network" {
  name                = "kubernetes-network-${random_string.suffix.result}"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  network_offering_id = data.virakcloud_network_service_offerings.isolated.offerings[0].id
  type                = "Isolated"
  gateway             = "10.10.0.1"
  netmask             = "255.255.255.0"
}

resource "virakcloud_kubernetes_cluster" "example_cluster" {
  name                  = "example-k8s-${random_string.suffix.result}"
  zone_id               = data.virakcloud_zones.available.zones[0].id
  kubernetes_version_id = data.virakcloud_kubernetes_versions.available.versions[0].id
  service_offering_id   = data.virakcloud_instance_service_offerings.k8s.offerings[0].id
  ssh_key_id            = var.ssh_key_id
  network_id            = virakcloud_network.k8s_network.id
  cluster_size          = 1
}

resource "virakcloud_instance" "k8s_worker" {
  name                = "k8s-worker-${random_string.suffix.result}"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  service_offering_id = data.virakcloud_instance_service_offerings.k8s.offerings[0].id
  vm_image_id         = data.virakcloud_instance_images.k8s.images[0].id
  network_ids         = [virakcloud_network.k8s_network.id]

  depends_on = [virakcloud_kubernetes_cluster.example_cluster]
}
