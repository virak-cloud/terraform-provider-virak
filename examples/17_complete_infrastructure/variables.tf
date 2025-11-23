variable "virak_token" {
  type        = string
  sensitive   = true
  description = "Your Virak Cloud API Token."
}


variable "vm_service_offering_name" {
  type        = string
  description = "The name of the service offering (e.g., 'Standard-1') for your VMs."
  default     = "Standard-1"
}

variable "vm_image_name" {
  type        = string
  description = "The name of the VM Image to deploy (e.g., 'Ubuntu 22.04')."
  default     = "Ubuntu 22.04"
}