output "port_forwarding_rule_id" {
  value = virakcloud_port_forwarding_rule.example.id
}

output "public_ip_address" {
  value = virakcloud_public_ip_association.example.ip_address
}

output "instance_id" {
  value = virakcloud_instance.example.id
}

