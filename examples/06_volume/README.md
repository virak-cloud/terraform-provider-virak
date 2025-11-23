# Volume Example

This example demonstrates how to create a volume first, then create an instance that can potentially be associated with that volume using the Virak Cloud Terraform provider.

## Environment Variables

The following environment variables are used to configure the Virak Cloud provider:

- `VIRAKCLOUD_TOKEN`: Required API token for authentication with the Virak Cloud API.

Set the environment variables using the following commands:

```bash
export VIRAKCLOUD_TOKEN="your-api-token-here"
```

These environment variables are used via the `env()` function in the variable defaults defined in `variables.tf`.

## Prerequisites

- Virak Cloud account with API access
- Terraform installed
- Virak Cloud provider configured

## What this example creates

- A 25GB volume (created first)
- An instance with a public network (created after the volume)
- Uses the first available zone, instance offering, image, and volume offering from your account

## Usage

1. Update the provider configuration in `main.tf` with your actual token.

2. Initialize Terraform:
   ```bash
   terraform init
   ```

3. Plan the deployment:
   ```bash
   terraform plan
   ```

4. Apply the configuration:
   ```bash
   terraform apply
   ```

5. Clean up when done:
   ```bash
   terraform destroy
   ```

## Outputs

- `instance_id`: The ID of the created instance
- `instance_status`: The status of the instance
- `volume_id`: The ID of the created volume
- `volume_status`: The status of the volume
- `volume_attached_instance_id`: The ID of the instance the volume is attached to (null since not attached)
- `volume_size`: The size of the volume in GB
- `volume_zone_id`: The zone ID where the volume is created

## Notes

- The volume is created first, followed by the instance
- The volume is created as a standalone volume (not attached to any instance)
- In Virak Cloud, volumes must be attached to instances to be used; standalone volumes are not bootable
- To attach the volume to the instance after creation, you would need to update the volume resource with `instance_id = virakcloud_instance.example.id`