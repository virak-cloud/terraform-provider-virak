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
  name      = "load-balancer-example-network"
  cidr      = "192.168.1.0/24"
  offering_id = var.network_offering_id
}

# Allocate a public IP for the load balancer
resource "virakcloud_public_ip_association" "lb" {
  network_id = virakcloud_network.example.id
}

# Create two instances for load balancing
resource "virakcloud_instance" "web1" {
  zone_id            = var.zone_id
  name               = "web-server-1"
  service_offering_id = var.instance_offering_id
  vm_image_id        = var.vm_image_id
  network_ids        = [virakcloud_network.example.id]
}

resource "virakcloud_instance" "web2" {
  zone_id            = var.zone_id
  name               = "web-server-2"
  service_offering_id = var.instance_offering_id
  vm_image_id        = var.vm_image_id
  network_ids        = [virakcloud_network.example.id]
}

# Create the load balancer rule
resource "virakcloud_load_balancer" "http" {
  zone_id      = var.zone_id
  network_id   = virakcloud_network.example.id
  public_ip_id = virakcloud_public_ip_association.lb.id
  name         = "http-load-balancer"
  algorithm    = "roundrobin"
  public_port  = 80
  private_port = 8080
}

# Attach the first instance to the load balancer
resource "virakcloud_load_balancer_backend" "web1" {
  zone_id             = var.zone_id
  network_id          = virakcloud_network.example.id
  load_balancer_id    = virakcloud_load_balancer.http.id
  instance_network_id = virakcloud_instance.web1.network_ids[0] # First network interface
}

# Attach the second instance to the load balancer
resource "virakcloud_load_balancer_backend" "web2" {
  zone_id             = var.zone_id
  network_id          = virakcloud_network.example.id
  load_balancer_id    = virakcloud_load_balancer.http.id
  instance_network_id = virakcloud_instance.web2.network_ids[0] # First network interface
}