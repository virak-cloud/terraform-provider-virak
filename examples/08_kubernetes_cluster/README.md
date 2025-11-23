# Virak Cloud Kubernetes Cluster Example

This example demonstrates how to create and manage a Kubernetes cluster using the Virak Cloud Terraform provider.

## Prerequisites

- Terraform >= 1.0
- Virak Cloud API token
- Access to a Virak Cloud zone with Kubernetes support

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

- `virakcloud_kubernetes_cluster.example_cluster`: A Kubernetes cluster with the latest available version
- `virakcloud_instance.k8s_worker`: An example worker instance (you can add more as needed)
- `virakcloud_network.k8s_network`: Network for Kubernetes workloads

## Data Sources Used

This example demonstrates the use of data sources to dynamically discover:
- Available Kubernetes versions
- Instance offerings and images
- Available zones

## Important Notes

- **Data Source Limitations**: Some data sources may return single items rather than arrays. Adjust the resource references accordingly based on your provider implementation.
- **Version Selection**: The example defaults to available versions. Uncomment specific version settings in main.tf to use particular Kubernetes versions.
- **Scaling**: Add more instances as needed for your Kubernetes workloads.
- **Networking**: The example creates a dedicated network for Kubernetes, but you may want to integrate with existing networks.

## Outputs

- `kubernetes_cluster_*`: Details about the created Kubernetes cluster
- `worker_instance_id`: ID of the example worker instance
- `k8s_network_id`: ID of the Kubernetes network
- `available_*`: Information about available resources discovered via data sources
