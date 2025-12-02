# Provider configuration
provider "virakcloud" {
  token = var.virakcloud_token
}

# Create a DNS domain
resource "virakcloud_dns_domain" "example_com" {
  domain = "example.com"
}

# A record pointing to an external IP (or you can reference an instance IP)
resource "virakcloud_dns_record" "www_a" {
  domain  = virakcloud_dns_domain.example_com.domain
  record  = "www"
  type    = "A"
  content = "192.168.1.10" # Replace with actual IP or reference instance
  ttl     = 3600
}

# CNAME record
resource "virakcloud_dns_record" "api_cname" {
  domain  = virakcloud_dns_domain.example_com.domain
  record  = "api"
  type    = "CNAME"
  content = "www.example.com"
  ttl     = 3600
}

# MX record for email
resource "virakcloud_dns_record" "mx_record" {
  domain   = virakcloud_dns_domain.example_com.domain
  record   = "@"
  type     = "MX"
  content  = "mail.example.com"
  priority = 10
  ttl      = 3600
}

# TXT record for verification
resource "virakcloud_dns_record" "txt_verification" {
  domain  = virakcloud_dns_domain.example_com.domain
  record  = "_dmarc"
  type    = "TXT"
  content = "v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com"
  ttl     = 3600
}
