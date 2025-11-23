# Virak Cloud Instance Snapshot Example

This example demonstrates creating and managing instance snapshots using the Virak Cloud Terraform provider.

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

- `virakcloud_network.example`: Network for the instance
- `virakcloud_instance.example`: Instance to be snapshotted
- `virakcloud_snapshot.example`: Snapshot of the instance

## Snapshot Operations

This example demonstrates:
- Creating a snapshot of a running instance
- The snapshot captures the current state of the instance for backup/restore purposes
- Snapshots can be used to revert instances to previous states

## Notes

- Snapshots can only be created from instances with status "UP"
- The snapshot creation process may take several minutes
- Snapshots are immutable once created