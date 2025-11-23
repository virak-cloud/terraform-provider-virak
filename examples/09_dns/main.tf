# Provider configuration
provider "virakcloud" {
  token    = var.virakcloud_token
}

# Create a DNS domain
resource "virakcloud_dns_domain" "example_com" {
  name = "example.com"
}

# A record pointing to an external IP (or you can reference an instance IP)
resource "virakcloud_dns_record" "www_a" {
  domain_id = virakcloud_dns_domain.example_com.id
  name      = "www"
  type      = "A"
  content   = "192.168.1.10"  # Replace with actual IP or reference instance
  ttl       = 3600
}

# CNAME record
resource "virakcloud_dns_record" "api_cname" {
  domain_id = virakcloud_dns_domain.example_com.id
  name      = "api"
  type      = "CNAME"
  content   = "www.example.com"
  ttl       = 3600
}

# MX record for email
resource "virakcloud_dns_record" "mx_record" {
  domain_id = virakcloud_dns_domain.example_com.id
  name      = "@"
  type      = "MX"
  content   = "10 mail.example.com"
  ttl       = 3600
}

# TXT record for verification
resource "virakcloud_dns_record" "txt_verification" {
  domain_id = virakcloud_dns_domain.example_com.id
  name      = "_dmarc"
  type      = "TXT"
  content   = "\"v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com\""
  ttl       = 3600
}

# Example instance to demonstrate pointing DNS to infrastructure
resource "virakcloud_network" "dns_example_network" {
  name         = "dns-example-network"
  cidr         = "192.168.20.0/24"
  gateway      = "192.168.20.1"
  netmask      = "255.255.255.0"
  network_type = "L3"

  zone_id = data.virakcloud_zones.available.zone_id
}

resource "virakcloud_instance" "web_server" {
  name = "web-server"

  offering_id = data.virakcloud_instance_offerings.available.offering_id
  image_id    = data.virakcloud_instance_images.available.image_id
  zone_id     = data.virakcloud_zones.available.zone_id
}

# Note: In a real scenario, you'd get the instance's IP address
# and use it in the DNS record. Since the provider doesn't expose
# instance IPs, this shows the pattern you would follow.

# Data sources
data "virakcloud_zones" "available" {}

data "virakcloud_instance_offerings" "available" {}

data "virakcloud_instance_images" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}
