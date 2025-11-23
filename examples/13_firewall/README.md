# Firewall Rules Example

This example demonstrates how to create firewall rules for both IPv4 and IPv6 networks using the Virak Cloud Terraform provider.

## Resources Created

- **Network**: A private network to attach firewall rules to
- **IPv4 Firewall Rules**:
  - SSH access (TCP port 22) from anywhere
  - Web access (TCP ports 80-443) from anywhere
  - ICMP (ping) from anywhere
- **IPv6 Firewall Rule**:
  - SSH access (TCP port 22) from anywhere (IPv6)

## Usage

1. Set your Virak Cloud API token:
   ```bash
   export VIRAKCLOUD_TOKEN="your-api-token-here"
   ```

2. Update the `terraform.tfvars` file with your zone and network offering IDs:
   ```hcl
   zone_id = "your-zone-id"
   network_offering_id = "your-network-offering-id"
   ```

3. Initialize and apply:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

## Notes

- Firewall rules are attached to networks and control traffic flow
- Both IPv4 and IPv6 are supported
- Traffic types include `ingress` (incoming) and `egress` (outgoing)
- Supported protocols: `tcp`, `udp`, `icmp`
- For ICMP rules, use `icmp_type` and `icmp_code` (set to -1 for all)
- Port ranges are supported using `start_port` and `end_port`