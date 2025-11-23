output "network_id" {
  description = "The ID of the created network"
  value       = virakcloud_network.example.id
}

output "instance_id" {
  description = "The ID of the created instance"
  value       = virakcloud_instance.example.id
}

output "public_ip_id" {
  description = "The ID of the allocated public IP"
  value       = virakcloud_public_ip_association.example.id
}

output "public_ip_address" {
  description = "The allocated public IP address"
  value       = virakcloud_public_ip_association.example.ip_address
}

output "public_ip_with_nat_id" {
  description = "The ID of the public IP with Static NAT"
  value       = virakcloud_public_ip.with_nat.id
}

output "public_ip_with_nat_address" {
  description = "The public IP address with Static NAT"
  value       = virakcloud_public_ip.with_nat.ip_address
}