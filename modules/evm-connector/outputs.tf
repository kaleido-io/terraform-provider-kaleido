output "service_id" {
  value       = kaleido_platform_service.this.id
  description = "ID of the EVMConnector service."
}

output "stack_id" {
  value       = kaleido_platform_stack.this.id
  description = "ID of the EVMConnectorStack."
}

output "runtime_id" {
  value       = kaleido_platform_runtime.this.id
  description = "ID of the EVMConnector runtime."
}

output "submission_flow_name" {
  value       = kaleido_platform_connector_flow.submission.name
  description = "Name of the deployed submission connector flow."
}

output "query_flow_name" {
  value       = kaleido_platform_connector_flow.query.name
  description = "Name of the deployed query connector flow."
}

output "standard_api_name" {
  value       = kaleido_platform_connector_standard_api.evm.name
  description = "Name of the deployed EVM standard API."
}

output "stream_factories" {
  value = {
    block_events       = kaleido_platform_connector_stream_factory.block_events.id
    transaction_events = kaleido_platform_connector_stream_factory.transaction_events.id
  }
  description = "IDs of the deployed connector stream factories."
}

output "config_profiles" {
  value       = { for k, v in kaleido_platform_connector_config_profile.this : k => v.id }
  description = "Map of config-type name to deployed config profile ID."
}
