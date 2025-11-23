output "volume_id" {
  description = "The ID of the created volume"
  value       = virakcloud_volume.example.id
}

output "volume_status" {
  description = "The status of the volume"
  value       = virakcloud_volume.example.status
}

output "volume_attached_instance_id" {
  description = "The ID of the instance the volume is attached to (null since volume is standalone)"
  value       = virakcloud_volume.example.attached_instance_id
}

output "volume_size" {
  description = "The size of the volume in GB"
  value       = 25
}

output "volume_zone_id" {
  description = "The zone ID where the volume is created"
  value       = virakcloud_volume.example.zone_id
}