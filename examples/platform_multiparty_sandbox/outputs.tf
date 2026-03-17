output "environment_id" {
  value = kaleido_platform_environment.env_0.id
}

output "contracts_service_id" {
  value = kaleido_platform_service.cms_0.id
}

output "ff_namespaces" {
  value = {for member in var.members : member => kaleido_platform_service.member_firefly[member].name}
}

output "ff_signing_keys" {
  value = {for member in var.members : member => kaleido_platform_kms_key.org_keys[member].address}
}

output "ff_service_ids" {
  value = {for member in var.members : member => kaleido_platform_service.member_firefly[member].id}
}