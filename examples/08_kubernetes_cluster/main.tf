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

# Kubernetes cluster example
resource "virakcloud_kubernetes_cluster" "example_cluster" {
  name = "example-k8s-cluster"

  # Use data source to get available Kubernetes version
  # version = data.virakcloud_kubernetes_versions.available.versions[0]

  # Uncomment to specify a specific version
  # version = "1.27.0"
}

# Example instance for Kubernetes workloads
resource "virakcloud_instance" "k8s_worker" {
  name = "k8s-worker-node"

  # Use data sources to get available offerings and images
  offering_id = data.virakcloud_instance_offerings.available.offering_id
  image_id    = data.virakcloud_instance_images.available.image_id
  zone_id     = data.virakcloud_zones.available.zone_id

  depends_on = [virakcloud_kubernetes_cluster.example_cluster]
}

# Network for the Kubernetes worker instances
resource "virakcloud_network" "k8s_network" {
  name         = "kubernetes-network"
  cidr         = "192.168.10.0/24"
  gateway      = "192.168.10.1"
  netmask      = "255.255.255.0"
  network_type = "L3"

  zone_id = data.virakcloud_zones.available.zone_id
}

# Data sources for Kubernetes versions, instance offerings, and zones
data "virakcloud_kubernetes_versions" "available" {}

data "virakcloud_instance_offerings" "available" {}

data "virakcloud_instance_images" "available" {}

data "virakcloud_zones" "available" {}
