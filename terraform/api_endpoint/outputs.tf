output "integration_id" {
  value       = aws_api_gateway_integration.integration.id
  description = "The integration ID, useful for deployment depends_on"
}