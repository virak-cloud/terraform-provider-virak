output "available_zones" {
  description = "Information about available zones"
  value       = data.virakcloud_zones.available
}

output "available_instance_offerings" {
  description = "Available instance offerings discovered via data source"
  value       = data.virakcloud_instance_offerings.available
}

output "available_instance_images" {
  description = "Available instance images discovered via data source"
  value       = data.virakcloud_instance_images.available
}

output "available_volume_offerings" {
  description = "Available volume offerings discovered via data source"
  value       = data.virakcloud_volume_offerings.available
}

output "available_network_offerings" {
  description = "Available network offerings discovered via data source"
  value       = data.virakcloud_network_offerings.available
}

output "available_kubernetes_versions" {
  description = "Available Kubernetes versions discovered via data source"
  value       = data.virakcloud_kubernetes_versions.available
}

output "selected_zone_id" {
  description = "The zone ID selected for resource deployment"
  value       = data.virakcloud_zones.available.zone_id
}

output "selected_offering_id" {
  description = "The instance offering ID selected for deployment"
  value       = data.virakcloud_instance_offerings.available.offering_id
}

output "selected_image_id" {
  description = "The instance image ID selected for deployment"
  value       = data.virakcloud_instance_images.available.image_id
}

output "infrastructure_summary" {
  description = "Summary of created infrastructure using discovered resources"
  value = {
    instance = {
      id         = virakcloud_instance.data_source_instance.id
      name       = virakcloud_instance.data_source_instance.name
      offering   = data.virakcloud_instance_offerings.available.offering_id
      image      = data.virakcloud_instance_images.available.image_id
    }
    network = {
      id   = virakcloud_network.data_source_network.id
      name = virakcloud_network.data_source_network.name
      cidr = virakcloud_network.data_source_network.cidr
    }
    volume = {
      id     = virakcloud_volume.data_source_volume.id
      name   = virakcloud_volume.data_source_volume.name
      size   = virakcloud_volume.data_source_volume.size
      offering = data.virakcloud_volume_offerings.available.offering_id
    }
    kubernetes = {
      id      = virakcloud_kubernetes_cluster.data_source_k8s.id
      name    = virakcloud_kubernetes_cluster.data_source_k8s.name
      version = virakcloud_kubernetes_cluster.data_source_k8s.version
    }
  }
}

output "resource_ids" {
  description = "All created resource IDs for reference"
  value = {
    # No resources created in this simplified example - only data sources
  }
}
