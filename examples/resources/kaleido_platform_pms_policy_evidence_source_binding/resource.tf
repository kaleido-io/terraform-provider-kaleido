# Evidence gathered by asking the members of an identity list version to approve or reject
resource "kaleido_platform_pms_policy_evidence_source_binding" "approvers" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  policy      = kaleido_platform_pms_policy.dual_approval.id
  name        = "approvers"
  type        = "approval"

  approval = {
    approval = {
      payload_type             = "TypedDataV4"
      payload_template_jsonata = "$.decision.approve"
    }
    rejection = {
      payload_type             = "TypedDataV4"
      payload_template_jsonata = "$.decision.reject"
    }
    identity_list_version = {
      id      = kaleido_platform_pms_identity_list.treasury_ops.applied_version_id
      version = kaleido_platform_pms_identity_list.treasury_ops.applied_version
    }
  }
}

# Evidence mapped directly out of the transaction input
resource "kaleido_platform_pms_policy_evidence_source_binding" "documents" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  policy      = kaleido_platform_pms_policy.dual_approval.id
  name        = "documents"
  type        = "attachment"

  attachment = {
    payload_jsonata     = "$.input.document"
    attestation_jsonata = "$.input.signature"
  }
}
