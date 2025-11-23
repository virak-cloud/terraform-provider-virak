output "kubernetes_cluster_id" {
  description = "The unique ID of the Kubernetes cluster"
  value       = virakcloud_kubernetes_cluster.example_cluster.id
}

output "kubernetes_cluster_name" {
  description = "The name of the Kubernetes cluster"
  value       = virakcloud_kubernetes_cluster.example_cluster.name
}

output "kubernetes_cluster_version" {
  description = "The Kubernetes version of the cluster"
  value       = virakcloud_kubernetes_cluster.example_cluster.version
}

output "kubernetes_cluster_status" {
  description = "The current status of the Kubernetes cluster"
  value       = virakcloud_kubernetes_cluster.example_cluster.status
}

output "worker_instance_id" {
  description = "The ID of the worker instance"
  value       = virakcloud_instance.k8s_worker.id
}

output "k8s_network_id" {
  description = "The ID of the Kubernetes network"
  value       = virakcloud_network.k8s_network.id
}

output "available_kubernetes_versions" {
  description = "List of available Kubernetes versions"
  value       = data.virakcloud_kubernetes_versions.available.versions
}

output "available_instance_offerings" {
  description = "Available instance offerings"
  value       = data.virakcloud_instance_offerings.available
}

output "available_zones" {
  description = "Available zones in Virak Cloud"
  value       = data.virakcloud_zones.available
}
