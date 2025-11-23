# Task 2: Establish a private network
resource "virakcloud_network" "private_net" {
  name = "my-private-network"

  # Use the ID found in data.tf
  network_offering_id = data.virakcloud_network_offering.l3_isolated.id

  # Define your private network's IP range
  gateway = "10.0.1.1"
  netmask = "255.255.255.0"
}

# Task 4: Assign a static IPv4 address to the private network
resource "virakcloud_public_ip_association" "net_public_ip" {
  # This implicitly depends on the network being created first
  network_id = virakcloud_network.private_net.id
}

# Task 1: Create the first VM
# Task 3: Connect the VM to the private network
resource "virakcloud_instance" "vm1" {
  name = "web-server-01"

  # Use IDs from data.tf
  service_offering_id = data.virakcloud_service_offering.vm_plan.id
  vm_image_id         = data.virakcloud_vm_image.os.id

  # This connects the VM to the network, creating a dependency
  network_ids = [virakcloud_network.private_net.id]
}

# Task 5: Create a second VM
# Task 6: Connect it to the same private network
resource "virakcloud_instance" "vm2" {
  name = "app-server-01"

  service_offering_id = data.virakcloud_service_offering.vm_plan.id
  vm_image_id         = data.virakcloud_vm_image.os.id

  # Connects this VM to the *same* network
  network_ids = [virakcloud_network.private_net.id]
}