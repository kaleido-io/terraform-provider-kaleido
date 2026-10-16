# A policy is a container. Its versions are created with kaleido_platform_pms_policy_version,
# and the bindings a version's definition resolves against are declared with the
# kaleido_platform_pms_policy_identity_list_binding,
# kaleido_platform_pms_policy_evidence_source_binding and
# kaleido_platform_pms_policy_output_formatter_binding resources.
resource "kaleido_platform_pms_policy" "dual_approval" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  name        = "dual_approval"
  description = "Requires two approvals from the treasury operations list"
}
