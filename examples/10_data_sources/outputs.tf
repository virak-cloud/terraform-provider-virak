output "available_zones" {
  description = "Information about available zones"
  value       = data.virakcloud_zones.available.zones
}

output "available_instance_offerings" {
  description = "Available instance offerings discovered via data source"
  value       = data.virakcloud_instance_service_offerings.available.offerings
}

output "available_instance_images" {
  description = "Available instance images discovered via data source"
  value       = data.virakcloud_instance_images.available.images
}

output "available_volume_offerings" {
  description = "Available volume offerings discovered via data source"
  value       = data.virakcloud_volume_service_offerings.available.offerings
}

output "available_network_offerings" {
  description = "Available network offerings discovered via data source"
  value       = data.virakcloud_network_service_offerings.available.offerings
}

output "available_kubernetes_versions" {
  description = "Available Kubernetes versions discovered via data source"
  value       = data.virakcloud_kubernetes_versions.available.versions
}

output "selected_zone_id" {
  description = "The zone ID selected for resource deployment"
  value       = data.virakcloud_zones.available.zones[0].id
}

output "selected_offering_id" {
  description = "The instance offering ID selected for deployment"
  value       = data.virakcloud_instance_service_offerings.available.offerings[0].id
}

output "selected_image_id" {
  description = "The instance image ID selected for deployment"
  value       = data.virakcloud_instance_images.available.images[0].id
}

output "infrastructure_summary" {
  description = "Summary of created infrastructure using discovered resources"
  value = {
    zone_name          = data.virakcloud_zones.available.zones[0].name
    service_offering   = data.virakcloud_instance_service_offerings.available.offerings[0].name
    volume_offering    = data.virakcloud_volume_service_offerings.available.offerings[0].name
    network_offering   = data.virakcloud_network_service_offerings.available.offerings[0].name
    kubernetes_version = data.virakcloud_kubernetes_versions.available.versions[0].version
  }
}
