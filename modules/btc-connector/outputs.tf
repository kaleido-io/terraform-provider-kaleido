output "service_id" {
  value       = kaleido_platform_service.this.id
  description = "ID of the BTCConnector service."
}

output "stack_id" {
  value       = kaleido_platform_stack.this.id
  description = "ID of the BTCConnectorStack."
}

output "runtime_id" {
  value       = kaleido_platform_runtime.this.id
  description = "ID of the BTCConnector runtime."
}

output "submission_flow_name" {
  value       = kaleido_platform_connector_flow.submission.name
  description = "Name of the deployed submission connector flow."
}

output "standard_api_name" {
  value       = kaleido_platform_connector_standard_api.bitcoin.name
  description = "Name of the deployed Bitcoin standard API."
}

output "stream_factories" {
  value = {
    transaction_events = kaleido_platform_connector_stream_factory.transaction_events.id
  }
  description = "IDs of the deployed connector stream factories."
}

output "config_profiles" {
  value       = { for k, v in kaleido_platform_connector_config_profile.this : k => v.id }
  description = "Map of config-type name to deployed config profile ID."
}
