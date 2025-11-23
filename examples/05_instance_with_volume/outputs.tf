output "instance_id" {
  description = "The ID of the created instance"
  value       = virakcloud_instance.instance_with_volume.id
}

output "instance_name" {
  description = "The name of the created instance"
  value       = virakcloud_instance.instance_with_volume.name
}

output "instance_status" {
  description = "The status of the created instance"
  value       = virakcloud_instance.instance_with_volume.status
}

output "instance_password" {
  description = "The password for the instance (sensitive)"
  value       = virakcloud_instance.instance_with_volume.password
  sensitive   = true
}

output "instance_ip" {
  description = "The IP address of the instance"
  value       = virakcloud_instance.instance_with_volume.ip
}

output "volume_id" {
  description = "The ID of the created volume"
  value       = virakcloud_instance.instance_with_volume.volumes[0].id
}

output "volume_name" {
  description = "The name of the created volume"
  value       = virakcloud_instance.instance_with_volume.volumes[0].name
}