variable "virakcloud_token" {
  description = "VirakCloud API token"
  type        = string
  sensitive   = true
}

variable "ssh_key_id" {
  description = "Existing VirakCloud SSH key ID used for Kubernetes cluster access"
  type        = string
}
