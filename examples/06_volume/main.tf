provider "virakcloud" {
  token    = var.virakcloud_token
  verbose  = true
}

# Data Sources - Fetch available zones
data "virakcloud_zones" "available" {}

# Data Sources - Fetch available instance service offerings
data "virakcloud_instance_service_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Data Sources - Fetch available instance images
data "virakcloud_instance_images" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Data Sources - Fetch available volume service offerings
data "virakcloud_volume_service_offerings" "available" {
  zone_id = data.virakcloud_zones.available.zones[0].id
}

# Volume Resource - Create a volume first (without attachment)
resource "virakcloud_volume" "example" {
  name                = "createdbyterraformvolume2"
  zone_id             = data.virakcloud_zones.available.zones[0].id
  service_offering_id = "01h75a4gz97t2qbghm9xtr8ytd" # g01-01-025-250-v3
  size                = 25
  # instance_id not set, so volume is created standalone
}

