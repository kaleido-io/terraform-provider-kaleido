# The versions of the submission connector flow the connector service stores, newest first.
data "kaleido_platform_connector_template_versions" "submission" {
  environment = kaleido_platform_environment.env.id
  service     = kaleido_platform_service.evm_connector.id
  kind        = "connector_flow"
  name        = "submission"
}

# Keep the flow on the newest version: a connector upgrade that brings a newer template shows up
# as an upgrade in the next plan.
resource "kaleido_platform_connector_flow" "submission" {
  environment = kaleido_platform_environment.env.id
  service     = kaleido_platform_service.evm_connector.id
  name        = "submission"
  version     = data.kaleido_platform_connector_template_versions.submission.latest
  # config_profiles = { ... }
}

# Or just report what is available without changing anything.
output "submission_supported_versions" {
  value = [for v in data.kaleido_platform_connector_template_versions.submission.versions : v.version if v.supported]
}
