output "network_id" {
  description = "The ID of the created network"
  value       = virakcloud_network.example.id
}

output "public_ip_id" {
  description = "The ID of the allocated public IP"
  value       = virakcloud_public_ip_association.lb.id
}

output "public_ip_address" {
  description = "The allocated public IP address"
  value       = virakcloud_public_ip_association.lb.ip_address
}

output "web1_instance_id" {
  description = "The ID of the first web server instance"
  value       = virakcloud_instance.web1.id
}

output "web2_instance_id" {
  description = "The ID of the second web server instance"
  value       = virakcloud_instance.web2.id
}

output "load_balancer_id" {
  description = "The ID of the load balancer rule"
  value       = virakcloud_load_balancer.http.id
}

output "load_balancer_status" {
  description = "The status of the load balancer rule"
  value       = virakcloud_load_balancer.http.status
}

output "backend_web1_id" {
  description = "The ID of the first backend assignment"
  value       = virakcloud_load_balancer_backend.web1.id
}

output "backend_web2_id" {
  description = "The ID of the second backend assignment"
  value       = virakcloud_load_balancer_backend.web2.id
}