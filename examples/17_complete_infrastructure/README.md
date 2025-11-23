# Complete Infrastructure Example

This example demonstrates creating a private network, associating a public IPv4 address to it, and provisioning two VMs attached to that network.

Usage:
- Set TF_VAR_virak_token and TF_VAR_virak_zone_id environment variables or create a terraform.tfvars with `virak_token` and `virak_zone_id`.
- Initialize and run:
  terraform init && terraform apply

Note: The provider block in this example uses `var.virak_token` and `var.virak_zone_id` to demonstrate provider-level zone_id support.