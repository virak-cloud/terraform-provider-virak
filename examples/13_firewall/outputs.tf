output "network_id" {
  description = "The ID of the created network"
  value       = virakcloud_network.example.id
}

output "ssh_firewall_rule_ipv4_id" {
  description = "The ID of the SSH IPv4 firewall rule"
  value       = virakcloud_firewall_rule.ssh_ipv4.id
}

output "web_firewall_rule_ipv4_id" {
  description = "The ID of the web IPv4 firewall rule"
  value       = virakcloud_firewall_rule.web_ipv4.id
}

output "ssh_firewall_rule_ipv6_id" {
  description = "The ID of the SSH IPv6 firewall rule"
  value       = virakcloud_firewall_rule.ssh_ipv6.id
}

output "icmp_firewall_rule_ipv4_id" {
  description = "The ID of the ICMP IPv4 firewall rule"
  value       = virakcloud_firewall_rule.icmp_ipv4.id
}