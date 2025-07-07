output "telemetry_id" {
  description = "The unique identifier of the telemetry configuration"
  value       = data.ngc_cloud_function_telemetry.example.id
}

output "telemetry_name" {
  description = "The name of the telemetry configuration"
  value       = data.ngc_cloud_function_telemetry.example.name
}

output "telemetry_endpoint" {
  description = "The telemetry endpoint URL"
  value       = data.ngc_cloud_function_telemetry.example.endpoint
}

output "telemetry_protocol" {
  description = "The protocol used for telemetry"
  value       = data.ngc_cloud_function_telemetry.example.protocol
}

output "telemetry_provider" {
  description = "The telemetry provider"
  value       = data.ngc_cloud_function_telemetry.example.telemetry_provider
}

output "telemetry_types" {
  description = "The telemetry data types"
  value       = data.ngc_cloud_function_telemetry.example.types
}

output "telemetry_created_at" {
  description = "The timestamp when the telemetry was created"
  value       = data.ngc_cloud_function_telemetry.example.created_at
}
