# Resolves the "treasuryOperations" attester label declared in the policy definition
# to a specific version of an identity list. Re-pointing at a newer version is an
# in-place update; changing the label replaces the binding.
resource "kaleido_platform_pms_policy_identity_list_binding" "treasury_ops" {
  environment              = kaleido_platform_environment.env_0.id
  service                  = kaleido_platform_service.pms_0.id
  policy                   = kaleido_platform_pms_policy.dual_approval.id
  attester_label           = "treasuryOperations"
  identity_list_version_id = kaleido_platform_pms_identity_list.treasury_ops.applied_version_id
}
