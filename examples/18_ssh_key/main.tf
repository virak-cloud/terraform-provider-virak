provider "virakcloud" {
  token    = var.virakcloud_token
}

resource "virakcloud_ssh_key" "example" {
  name      = "my-ssh-key"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC..."
}

