output "snapshot_id" {
  description = "The ID of the created snapshot"
  value       = virakcloud_snapshot.example.id
}

output "snapshot_name" {
  description = "The name of the snapshot"
  value       = virakcloud_snapshot.example.name
}

output "snapshot_status" {
  description = "The status of the snapshot"
  value       = virakcloud_snapshot.example.status
}

output "snapshot_created_at" {
  description = "The creation timestamp of the snapshot"
  value       = virakcloud_snapshot.example.created_at
}

output "instance_id" {
  description = "The ID of the instance that was snapshotted"
  value       = virakcloud_instance.example.id
}

output "network_id" {
  description = "The ID of the network created for the example"
  value       = virakcloud_network.example.id
}