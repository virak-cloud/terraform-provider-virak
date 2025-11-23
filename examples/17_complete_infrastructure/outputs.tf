output "private_network_id" {
  description = "The ID of the new private network."
  value       = virakcloud_network.private_net.id
}

output "assigned_public_ip" {
  description = "The static IPv4 address assigned to your network."
  value       = virakcloud_public_ip_association.net_public_ip.ip_address
}

output "vm1_details" {
  description = "Details for the first VM (web-server-01)."
  value = {
    id       = virakcloud_instance.vm1.id
    status   = virakcloud_instance.vm1.status
    username = virakcloud_instance.vm1.username
    password = virakcloud_instance.vm1.password
  }
  sensitive = true
}

output "vm2_details" {
  description = "Details for the second VM (app-server-01)."
  value = {
    id       = virakcloud_instance.vm2.id
    status   = virakcloud_instance.vm2.status
    username = virakcloud_instance.vm2.username
    password = virakcloud_instance.vm2.password
  }
  sensitive = true
}