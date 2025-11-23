output "l2_network_id" {
  description = "The ID of the created L2 network"
  value       = virakcloud_network.l2_network.id
}

output "l2_network_name" {
  description = "The name of the L2 network"
  value       = virakcloud_network.l2_network.name
}

output "l2_network_status" {
  description = "The status of the L2 network"
  value       = virakcloud_network.l2_network.status
}

# output "l3_network_id" {
#   description = "The ID of the created L3 network"
#   value       = virakcloud_network.l3_network.id
# }

# output "l3_network_name" {
#   description = "The name of the L3 network"
#   value       = virakcloud_network.l3_network.name
# }

# output "l3_network_gateway" {
#   description = "The gateway of the L3 network"
#   value       = virakcloud_network.l3_network.gateway
# }

# output "l3_network_netmask" {
#   description = "The netmask of the L3 network"
#   value       = virakcloud_network.l3_network.netmask
# }

# output "l3_network_status" {
#   description = "The status of the L3 network"
#   value       = virakcloud_network.l3_network.status
# }