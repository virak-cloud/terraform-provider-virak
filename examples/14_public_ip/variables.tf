variable "zone_id" {
  description = "The ID of the zone where resources will be created"
  type        = string
}

variable "network_offering_id" {
  description = "The ID of the network service offering"
  type        = string
}

variable "instance_offering_id" {
  description = "The ID of the instance service offering"
  type        = string
}

variable "vm_image_id" {
  description = "The ID of the VM image"
  type        = string
}