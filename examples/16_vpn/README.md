# Virak Cloud Network VPN Example

This example demonstrates enabling VPN on a Virak Cloud network using the Virak Cloud Terraform provider.

## Prerequisites

- Virak Cloud API token
- Terraform >= 1.0

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

# Plan the deployment
terraform plan

# Apply the configuration
terraform apply
```

## Resources Created

This example creates:

1. A Virak Cloud network (Isolated type)
2. VPN configuration for the network with authentication credentials

## Outputs

- `network_id`: The ID of the created network
- `network_name`: The name of the network
- `vpn_enabled`: Whether VPN is enabled
- `vpn_ip_address`: The VPN IP address
- `vpn_preshared_key`: The VPN preshared key (sensitive)