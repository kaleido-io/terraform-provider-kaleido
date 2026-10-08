# The policy is created as an empty container, so that the bindings its definition
# depends on exist before the first version is cut. Terraform orders the version after
# the bindings it references.
resource "kaleido_platform_pms_policy" "dual_approval" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  name        = "dual_approval"
  description = "Requires two approvals from the treasury operations list"
}

resource "kaleido_platform_pms_policy_identity_list_binding" "treasury_operations" {
  environment              = kaleido_platform_environment.env_0.id
  service                  = kaleido_platform_service.pms_0.id
  policy                   = kaleido_platform_pms_policy.dual_approval.id
  attester_label           = "treasuryOperations"
  identity_list_version_id = kaleido_platform_pms_identity_list.treasury_ops.applied_version_id
}

resource "kaleido_platform_pms_policy_version" "v1" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  policy      = kaleido_platform_pms_policy.dual_approval.id
  name        = "v1"

  definition_yaml = file("${path.module}/policies/dual_approval.yaml")

  depends_on = [kaleido_platform_pms_policy_identity_list_binding.treasury_operations]
}
