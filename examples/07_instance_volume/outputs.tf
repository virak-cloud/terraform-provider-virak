output "instance_id" {
  description = "The ID of the created instance"
  value       = virakcloud_instance.example.id
}

output "instance_name" {
  description = "The name of the created instance"
  value       = virakcloud_instance.example.name
}

output "instance_status" {
  description = "The status of the created instance"
  value       = virakcloud_instance.example.status
}

output "instance_password" {
  description = "The password for the instance (sensitive)"
  value       = virakcloud_instance.example.password
  sensitive   = true
}

output "volume_ids" {
  description = "The IDs of the volumes created with the instance"
  value       = virakcloud_instance.example.volume_ids
}

output "volume_id" {
  description = "The ID of the created volume"
  value       = virakcloud_volume.example.id
}

output "volume_name" {
  description = "The name of the created volume"
  value       = virakcloud_volume.example.name
}

output "volume_status" {
  description = "The status of the created volume"
  value       = virakcloud_volume.example.status
}

output "attached_instance_id" {
  description = "The ID of the instance the volume is attached to"
  value       = virakcloud_volume.example.attached_instance_id
}