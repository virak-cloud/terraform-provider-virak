# Virak Cloud Advanced Volumes Example

This example demonstrates advanced volume management patterns, including multiple volume types, sizing strategies, and volume lifecycle management.

## Prerequisites

- Terraform >= 1.0
- Virak Cloud API token
- Random provider for bucket naming

## Environment Variables

The following environment variables are used to configure the Virak Cloud provider:

- `VIRAKCLOUD_TOKEN`: Required API token for authentication with the Virak Cloud API.

Set the environment variables using the following commands:

```bash
export VIRAKCLOUD_TOKEN="your-api-token-here"
```

These environment variables are used via the `env()` function in the variable defaults defined in `variables.tf`.

## Volume Types Demonstrated

### Purpose-Based Sizing
- **Root Volume (50GB)**: For OS and applications, larger for system stability
- **Data Volume (100GB)**: For databases and persistent data storage
- **Backup Volume (200GB)**: For backups, archives, and snapshots
- **Staging Volume (20GB)**: Temporary space for development/testing
- **Temporary Volume (10GB)**: Ephemeral storage with lifecycle management

### Volume Management Patterns

#### Multi-Volume Creation
Multiple volumes created for different purposes:
- Separate OS, data, and backup concerns
- Improved isolation and backup strategies
- Different performance characteristics per volume

#### Standalone Volumes
Volumes created for future use:
- Future expansion planning
- Backup and recovery scenarios
- Volume lifecycle management

#### Lifecycle Management
- `create_before_destroy`: Ensures service continuity during volume recreation
- Volume lifecycle patterns for different use cases

## Usage

```bash
# Initialize Terraform
terraform init

# Review the plan
terraform plan

# Apply the configuration
terraform apply

# View detailed volume information
terraform output volumes_summary

# View total capacity
terraform output total_volume_capacity

# Clean up when done
terraform destroy
```

## Important Concepts

### Volume Sizing Strategy
Different workloads require different storage sizes:
- **Development/Testing**: Smaller volumes, frequently recreated
- **Production Databases**: Larger volumes for data growth
- **Backup Systems**: Even larger volumes for retention policies

### Data Persistence vs. Ephemeral Storage
- **Persistent Volumes**: Root, Data, Backup (survive instance termination)
- **Ephemeral Volumes**: Staging, Temporary (can be recreated)

### Volume Performance Considerations
Different volume types might have different:
- IOPS capabilities
- Throughput characteristics
- Cost structures
- Availability guarantees

### Volume Migration
The example shows concepts for volume lifecycle management:
- Using `lifecycle.replace_triggered_by` for automated recreation
- Creating migration target instances
- Managing volume transitions

## Best Practices Demonstrated

### Resource Organization
- Clear volume naming conventions
- Logical grouping of related resources
- Output structures for easy consumption

### Cost Optimization
- Right-sizing volumes for their intended use
- Considering volume lifecycle to avoid over-provisioning

### Operational Management
- Separate volumes for different data types
- Backup volume isolation
- Temporary volume lifecycle management

## Advanced Features

### Random Resource Naming
Uses `random_string` to generate unique bucket names, demonstrating:
- Avoiding naming conflicts
- Dynamic resource naming strategies

### Complex Output Structures
Provides structured outputs showing:
- Volume summaries
- Instance-volume relationships
- Network topology
- Total capacity calculations

## Real-World Applications

This pattern is commonly used for:
- **Database Systems**: Separating OS, data, and log volumes
- **Web Services**: Root volume for OS, separate volumes for logs/backups
- **Development Environments**: Staging volumes that can be easily reset
- **Migration Scenarios**: Moving data between instances during upgrades

## Outputs

- Volume inventory with sizes and IDs
- Instance-to-volume mapping
- Network configuration details
- Storage capacity totals
- Backup infrastructure (bucket) details
