output "volumes_summary" {
  description = "Summary of all created volumes"
  value = {
    root_volume = {
      id     = virakcloud_volume.root_volume.id
      name   = virakcloud_volume.root_volume.name
      size   = virakcloud_volume.root_volume.size
    }
    data_volume = {
      id     = virakcloud_volume.data_volume.id
      name   = virakcloud_volume.data_volume.name
      size   = virakcloud_volume.data_volume.size
    }
    backup_volume = {
      id     = virakcloud_volume.backup_volume.id
      name   = virakcloud_volume.backup_volume.name
      size   = virakcloud_volume.backup_volume.size
    }
    staging_volume = {
      id     = virakcloud_volume.staging_volume.id
      name   = virakcloud_volume.staging_volume.name
      size   = virakcloud_volume.staging_volume.size
    }
    temporary_volume = {
      id     = virakcloud_volume.temporary_volume.id
      name   = virakcloud_volume.temporary_volume.name
      size   = virakcloud_volume.temporary_volume.size
    }
  }
}

output "instance_volumes" {
  description = "Volumes attached to each instance"
  value = {
    volume_host = {
      instance_id = virakcloud_instance.volume_host.id
      attached_volumes = [
        virakcloud_volume.root_volume.id,
        virakcloud_volume.data_volume.id,
        virakcloud_volume.backup_volume.id,
        virakcloud_volume.temporary_volume.id
      ]
    }
    migration_target = {
      instance_id = virakcloud_instance.migration_target.id
      attached_volumes = []
    }
  }
}

output "network_details" {
  description = "Network configuration for volumes infrastructure"
  value = {
    network_id = virakcloud_network.volume_example_network.id
    name       = virakcloud_network.volume_example_network.name
    gateway    = virakcloud_network.volume_example_network.gateway
    netmask    = virakcloud_network.volume_example_network.netmask
    type       = virakcloud_network.volume_example_network.type
    instances_attached = [
      virakcloud_instance.volume_host.id,
      virakcloud_instance.migration_target.id
    ]
  }
}

output "total_volume_capacity" {
  description = "Total volume storage capacity allocated"
  value = (
    virakcloud_volume.root_volume.size +
    virakcloud_volume.data_volume.size +
    virakcloud_volume.backup_volume.size +
    virakcloud_volume.staging_volume.size +
    virakcloud_volume.temporary_volume.size
  )
}
