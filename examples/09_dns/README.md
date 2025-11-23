# Virak Cloud DNS Management Example

This example demonstrates how to create and manage DNS domains and records using the Virak Cloud Terraform provider, including different record types and best practices.

## Prerequisites

- Terraform >= 1.0
- Virak Cloud API token
- Access to a Virak Cloud zone

## Environment Variables

The following environment variables are used to configure the Virak Cloud provider:

- `VIRAKCLOUD_TOKEN`: Required API token for authentication with the Virak Cloud API.

Set the environment variables using the following commands:

```bash
export VIRAKCLOUD_TOKEN="your-api-token-here"
```

These environment variables are used via the `env()` function in the variable defaults defined in `variables.tf`.

## Usage

```bash
# Initialize Terraform
terraform init

# Review the plan
terraform plan

# Apply the configuration
terraform apply

# View outputs
terraform output

# Clean up when done
terraform destroy
```

## Resources Created

- `virakcloud_dns_domain.example_com`: DNS domain "example.com"
- `virakcloud_dns_record.www_a`: A record for "www.example.com" pointing to an IP address
- `virakcloud_dns_record.api_cname`: CNAME record for "api.example.com" pointing to "www.example.com"
- `virakcloud_dns_record.mx_record`: MX record for email delivery
- `virakcloud_dns_record.txt_verification`: TXT record for DMARC/email verification
- Example web server instance and network to demonstrate infrastructure integration

## DNS Record Types Demonstrated

### A Record
- Points a domain name to an IPv4 address
- TTL: 3600 seconds (1 hour)

### CNAME Record
- Creates an alias from one domain to another
- Used for subdomains pointing to the main domain

### MX Record
- Specifies mail servers for email delivery
- Priority-based routing (lower number = higher priority)

### TXT Record
- Stores arbitrary text data
- Used for domain verification, SPF, DKIM, DMARC policies

## Important Notes

- **IP Address Management**: The A record uses a placeholder IP (`192.168.1.10`). In production, you would:
  1. Deploy your infrastructure first
  2. Get the actual IP address from Virak Cloud
  3. Update the DNS record with the real IP
  4. Or use a data source if the provider exposes instance IPs

- **Domain Registration**: Don't forget to delegate your domain to Virak Cloud's nameservers at your domain registrar

- **TTL Values**: Adjust TTL values based on your needs:
  - Lower TTL for frequently changing records
  - Higher TTL for stable records (better performance)

- **Security**: Consider using DNSSEC for enhanced security when available

## Integration with Infrastructure

This example shows how to combine DNS management with infrastructure provisioning. In a real scenario, you might:

1. Deploy load balancers or instances
2. Retrieve their IP addresses
3. Create A records pointing to those IPs
4. Set up CNAME records for service discovery

## Outputs

- Domain and record details
- Example infrastructure information
- Structured output showing all DNS records created
