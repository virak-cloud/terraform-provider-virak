# Load Balancer Example

This example demonstrates how to create and manage load balancers with backend instance assignments using the Virak Cloud Terraform provider.

## Resources Created

- **Network**: A private network for all resources
- **Public IP**: An allocated public IP for the load balancer
- **Instances**: Two web server instances for load balancing
- **Load Balancer**: A load balancer rule with round-robin algorithm
- **Backend Assignments**: Two backend assignments connecting instances to the load balancer

## Architecture

```
Internet -> Public IP (80) -> Load Balancer -> Instance 1 (8080)
                                      -> Instance 2 (8080)
```

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

- Load balancers require a public IP address
- Backend instances must be in the same network as the load balancer
- The `instance_network_id` refers to the network interface ID of the instance
- Supported algorithms include `roundrobin`, `leastconn`, etc.
- Load balancer rules are immutable - changes require creating a new rule
- Backend assignments can be added/removed independently