# Instance with Private Network Example

This example demonstrates how to create a Virak Cloud instance attached to a private L3 network.

## Environment Variables

The following environment variables are used to configure the Virak Cloud provider:

- `VIRAKCLOUD_TOKEN`: Required API token for authentication with the Virak Cloud API.

Set the environment variables using the following commands:

```bash
export VIRAKCLOUD_TOKEN="your-api-token-here"
```

These environment variables are used via the `env()` function in the variable defaults defined in `variables.tf`.

## Resources Created

1. **Private L3 Network**: An isolated network with custom IP range (192.168.1.0/24)
2. **Instance**: A virtual machine attached to the private network

## Usage

1. **Initialize Terraform**:
   ```bash
   terraform init
   ```

2. **Plan the deployment**:
   ```bash
   terraform plan
   ```

3. **Apply the configuration**:
   ```bash
   terraform apply
   ```

4. **View outputs**:
   ```bash
   terraform output
   ```

5. **Clean up resources**:
   ```bash
   terraform destroy
   ```

## Configuration Notes

- The network is configured with a private IP range (192.168.1.0/24)
- The instance is automatically assigned an IP from this network
- The instance depends on the network being created first
- Random suffixes ensure unique resource names across multiple deployments