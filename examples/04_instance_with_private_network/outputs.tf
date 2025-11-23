output "network_id" {
  description = "The ID of the created private network"
  value       = virakcloud_network.private_network.id
}

output "network_name" {
  description = "The name of the created private network"
  value       = virakcloud_network.private_network.name
}

output "network_status" {
  description = "The status of the created private network"
  value       = virakcloud_network.private_network.status
}

output "instance_id" {
  description = "The ID of the created instance"
  value       = virakcloud_instance.private_network_instance.id
}

output "instance_name" {
  description = "The name of the created instance"
  value       = virakcloud_instance.private_network_instance.name
}

output "instance_status" {
  description = "The status of the created instance"
  value       = virakcloud_instance.private_network_instance.status
}

output "instance_ip" {
  description = "The IP address of the created instance"
  value       = virakcloud_instance.private_network_instance.ip
}

output "instance_password" {
  description = "The password for the instance (sensitive)"
  value       = virakcloud_instance.private_network_instance.password
  sensitive   = true
}