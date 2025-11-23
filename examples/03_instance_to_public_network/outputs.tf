output "instance_id" {
  description = "The ID of the created instance"
  value       = virakcloud_instance.public_network_instance.id
}

output "instance_name" {
  description = "The name of the created instance"
  value       = virakcloud_instance.public_network_instance.name
}

output "instance_status" {
  description = "The status of the created instance"
  value       = virakcloud_instance.public_network_instance.status
}

output "instance_password" {
  description = "The password for the instance (sensitive)"
  value       = virakcloud_instance.public_network_instance.password
  sensitive   = true
}