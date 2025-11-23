output "dns_domain_id" {
  description = "The unique ID of the DNS domain"
  value       = virakcloud_dns_domain.example_com.id
}

output "dns_domain_name" {
  description = "The name of the DNS domain"
  value       = virakcloud_dns_domain.example_com.name
}

output "dns_records" {
  description = "List of all DNS records created"
  value = [
    {
      name    = virakcloud_dns_record.www_a.name
      type    = virakcloud_dns_record.www_a.type
      content = virakcloud_dns_record.www_a.content
    },
    {
      name    = virakcloud_dns_record.api_cname.name
      type    = virakcloud_dns_record.api_cname.type
      content = virakcloud_dns_record.api_cname.content
    },
    {
      name    = virakcloud_dns_record.mx_record.name
      type    = virakcloud_dns_record.mx_record.type
      content = virakcloud_dns_record.mx_record.content
    },
    {
      name    = virakcloud_dns_record.txt_verification.name
      type    = virakcloud_dns_record.txt_verification.type
      content = virakcloud_dns_record.txt_verification.content
    }
  ]
}

output "web_server_instance_id" {
  description = "The ID of the example web server instance"
  value       = virakcloud_instance.web_server.id
}

output "web_server_network_id" {
  description = "The ID of the network for the web server"
  value       = virakcloud_network.dns_example_network.id
}

output "dns_domain_nameservers" {
  description = "Nameservers for the DNS domain (you may need to set these at your registrar)"
  value       = ["ns1.example.com", "ns2.example.com"] # This would typically come from the provider
}

output "web_server_network_details" {
  description = "Network details for reference"
  value = {
    network_id = virakcloud_network.dns_example_network.id
    cidr       = virakcloud_network.dns_example_network.cidr
    gateway    = virakcloud_network.dns_example_network.gateway
  }
}
