output "network_id" {
  description = "The ID of the created network"
  value       = virakcloud_network.vpn_network.id
}

output "network_name" {
  description = "The name of the network"
  value       = virakcloud_network.vpn_network.name
}

output "vpn_enabled" {
  description = "Whether VPN is enabled on the network"
  value       = virakcloud_network_vpn.example.enabled
}

output "vpn_ip_address" {
  description = "The VPN IP address"
  value       = virakcloud_network_vpn.example.ip_address
}

output "vpn_preshared_key" {
  description = "The VPN preshared key"
  value       = virakcloud_network_vpn.example.preshared_key
  sensitive   = true
}