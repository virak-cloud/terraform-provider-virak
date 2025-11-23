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
  name      = "firewall-example-network"
  cidr      = "192.168.1.0/24"
  offering_id = var.network_offering_id
}

# IPv4 Firewall Rule - Allow SSH from anywhere
resource "virakcloud_firewall_rule" "ssh_ipv4" {
  zone_id     = var.zone_id
  network_id  = virakcloud_network.example.id
  ip_version  = "ipv4"
  traffic_type = "ingress"
  protocol    = "tcp"
  ip_source   = "0.0.0.0/0"
  start_port  = 22
  end_port    = 22
}

# IPv4 Firewall Rule - Allow HTTP/HTTPS from anywhere
resource "virakcloud_firewall_rule" "web_ipv4" {
  zone_id     = var.zone_id
  network_id  = virakcloud_network.example.id
  ip_version  = "ipv4"
  traffic_type = "ingress"
  protocol    = "tcp"
  ip_source   = "0.0.0.0/0"
  start_port  = 80
  end_port    = 443
}

# IPv6 Firewall Rule - Allow SSH from anywhere (IPv6)
resource "virakcloud_firewall_rule" "ssh_ipv6" {
  zone_id       = var.zone_id
  network_id    = virakcloud_network.example.id
  ip_version    = "ipv6"
  traffic_type  = "ingress"
  protocol      = "tcp"
  ip_source     = "::/0"
  start_port    = 22
  end_port      = 22
}

# IPv4 Firewall Rule - Allow ICMP (ping)
resource "virakcloud_firewall_rule" "icmp_ipv4" {
  zone_id     = var.zone_id
  network_id  = virakcloud_network.example.id
  ip_version  = "ipv4"
  traffic_type = "ingress"
  protocol    = "icmp"
  ip_source   = "0.0.0.0/0"
  icmp_type   = -1
  icmp_code   = -1
}