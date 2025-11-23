# Virak Cloud Data Sources Example

This example demonstrates how to use Terraform data sources with the Virak Cloud provider to dynamically discover and select available resources like zones, instance offerings, images, and Kubernetes versions.

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

# Review the plan (will show discovered resources)
terraform plan

# Apply the configuration
terraform apply

# View outputs (all discovered and created resources)
terraform output

# Clean up when done
terraform destroy
```

## Data Sources Demonstrated

The example uses all available data sources:

- `data.virakcloud_zones`: Discovers available zones
- `data.virakcloud_instance_offerings`: Lists available instance types/offers
- `data.virakcloud_instance_images`: Lists available OS images
- `data.virakcloud_volume_offerings`: Lists available volume types
- `data.virakcloud_network_offerings`: Lists available network offerings
- `data.virakcloud_kubernetes_versions`: Lists available Kubernetes versions

## Resources Created

This example demonstrates data source usage without creating actual resources. The data sources show how to dynamically discover available:

- Zones
- Instance service offerings  
- Instance images
- Volume service offerings
- Network service offerings
- Kubernetes versions

## Benefits of Using Data Sources

### Dynamic Resource Selection
- Automatically selects available resources instead of hardcoding IDs
- Adapts to changes in available offerings without code updates
- Makes configurations more portable across different environments

### Discovery and Inspection
- Lists all available options before deployment
- Helps choose appropriate instance sizes, images, etc.
- Provides visibility into available resources

### Flexibility
- Easy to switch between different offerings by changing selection logic
- Supports filtering (when API supports it)
- Enables automated resource selection based on requirements

## Important Notes

- **Data Source Behavior**: Depending on your provider implementation, some data sources may return single items rather than arrays. Adjust the resource references accordingly.

- **Resource Dependencies**: The example shows how data sources can inform resource creation, making configurations more dynamic and adaptable.

- **Version Selection**: The Kubernetes cluster automatically selects the latest available version, but you can pin to specific versions by referencing array elements or using conditional logic.

- **Filtering**: The commented section shows how filtering might work (uncomment if your provider supports filtering parameters).

## Advanced Usage Patterns

### Resource Selection Strategies
```hcl
# Select smallest instance
offering_id = data.virakcloud_instance_offerings.available.offering_id

# Select by specific criteria (when filtering is supported)
data "virakcloud_instance_offerings" "filtered" {
  filter {
    name = "cpu_cores"
    values = ["2"]
  }
}

# Use latest image
image_id = data.virakcloud_instance_images.available.image_id

# Use latest K8s version
k8s_version = data.virakcloud_kubernetes_versions.available.versions[0]
```

### Environment-Specific Configurations
Data sources enable creating configurations that automatically adapt to different environments (development, staging, production) with varying available resources.

## Outputs

- Complete information about all discovered resources
- Infrastructure summary showing the selected resources
- All resource IDs for reference and management
