# Public IP Example

This example demonstrates how to allocate and manage public IPs in Virak Cloud using the Terraform provider.

## Resources Created

- **Network**: A private network for the public IP allocation
- **Instance**: A virtual machine instance
- **Public IP**: An allocated public IP address (without Static NAT)
- **Public IP with NAT**: An allocated public IP with Static NAT enabled to an instance

## Usage

1. Set your Virak Cloud API token:
   ```bash
   export VIRAKCLOUD_TOKEN="your-api-token-here"
   ```

2. Update the `terraform.tfvars` file with your zone and offering IDs:
   ```hcl
   zone_id = "your-zone-id"
   network_offering_id = "your-network-offering-id"
   instance_offering_id = "your-instance-offering-id"
   vm_image_id = "your-vm-image-id"
   ```

3. Initialize and apply:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

## Notes

- Public IPs are allocated within a network and can be associated with instances
- Static NAT allows direct access to an instance via the public IP
- Public IPs can be updated to enable/disable Static NAT by changing the `instance_id`
- The `ip_address` attribute contains the actual public IP address assigned