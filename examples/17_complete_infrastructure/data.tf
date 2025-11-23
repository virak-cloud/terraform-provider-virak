# Find the L3 (Isolated) Network Offering
data "virakcloud_network_offering" "l3_isolated" {
  filter {
    name   = "type"
    values = ["Isolated"]
  }
}

# Find the VM Service Offering (e.g., 'Standard-1')
data "virakcloud_service_offering" "vm_plan" {
  filter {
    name   = "name"
    values = [var.vm_service_offering_name]
  }
  filter {
    name   = "is_available"
    values = ["true"]
  }
}

# Find the VM Image (e.g., 'Ubuntu 22.04')
data "virakcloud_vm_image" "os" {
  filter {
    name   = "name"
    values = [var.vm_image_name]
  }
}