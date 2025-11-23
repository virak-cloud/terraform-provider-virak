variable "virakcloud_token" {
  type        = string
  description = "Virak Cloud API token"
  sensitive   = true
  default     = env("VIRAKCLOUD_TOKEN")
}


