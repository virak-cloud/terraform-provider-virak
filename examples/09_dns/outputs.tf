output "dns_domain_id" {
  description = "The unique ID of the DNS domain"
  value       = virakcloud_dns_domain.example_com.id
}

output "dns_domain_name" {
  description = "The name of the DNS domain"
  value       = virakcloud_dns_domain.example_com.domain
}

output "dns_records" {
  description = "List of all DNS records created"
  value = [
    {
      name    = virakcloud_dns_record.www_a.record
      type    = virakcloud_dns_record.www_a.type
      content = virakcloud_dns_record.www_a.content
    },
    {
      name    = virakcloud_dns_record.api_cname.record
      type    = virakcloud_dns_record.api_cname.type
      content = virakcloud_dns_record.api_cname.content
    },
    {
      name    = virakcloud_dns_record.mx_record.record
      type    = virakcloud_dns_record.mx_record.type
      content = virakcloud_dns_record.mx_record.content
    },
    {
      name    = virakcloud_dns_record.txt_verification.record
      type    = virakcloud_dns_record.txt_verification.type
      content = virakcloud_dns_record.txt_verification.content
    }
  ]
}
