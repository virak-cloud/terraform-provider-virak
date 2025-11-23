output "private_bucket_id" {
  description = "The ID of the created private bucket"
  value       = virakcloud_bucket.private_bucket.id
}

output "private_bucket_name" {
  description = "The name of the private bucket"
  value       = virakcloud_bucket.private_bucket.name
}

output "private_bucket_url" {
  description = "The URL of the private bucket"
  value       = virakcloud_bucket.private_bucket.url
}

output "private_bucket_status" {
  description = "The status of the private bucket"
  value       = virakcloud_bucket.private_bucket.status
}

output "public_bucket_id" {
  description = "The ID of the created public bucket"
  value       = virakcloud_bucket.public_bucket.id
}

output "public_bucket_name" {
  description = "The name of the public bucket"
  value       = virakcloud_bucket.public_bucket.name
}

output "public_bucket_url" {
  description = "The URL of the public bucket"
  value       = virakcloud_bucket.public_bucket.url
}

output "public_bucket_status" {
  description = "The status of the public bucket"
  value       = virakcloud_bucket.public_bucket.status
}