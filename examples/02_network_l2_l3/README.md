# Virak Cloud L2 and L3 Network Example

This example demonstrates creating L2 and L3 networks using the Virak Cloud Terraform provider.

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

# Review the plan
terraform plan

# Apply the configuration
terraform apply

# Clean up
terraform destroy
```

## Resources Created

- `virakcloud_network.l2_network`: L2 network (layer 2 isolation)
- `virakcloud_network.l3_network`: L3 network with gateway (192.168.1.1) and netmask (255.255.255.0)

## Network Types

- **L2 Network**: Provides layer 2 isolation without IP address management
- **L3 Network**: Provides layer 3 routing with configurable gateway and netmask

## Outputs

The configuration provides outputs for network IDs, names, and configuration details.