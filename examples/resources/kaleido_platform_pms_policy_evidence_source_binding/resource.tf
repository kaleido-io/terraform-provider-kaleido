# A binding ties one of the policy's evidence slots (evidence[].source) to an evidence
# source, plus the inputs that source needs from this policy.

# Approvals: both slots share one source and differ only in who is asked
resource "kaleido_platform_pms_policy_evidence_source_binding" "treasury_approval" {
  environment            = kaleido_platform_environment.env_0.id
  service                = kaleido_platform_service.pms_0.id
  policy                 = kaleido_platform_pms_policy.tiered_approval.id
  policy_evidence_source = "treasuryApproval"
  evidence_source_id     = kaleido_platform_pms_evidence_source.transfer_approval.id
  attesters              = "treasuryOperations" # an identity list binding label on this policy
}

resource "kaleido_platform_pms_policy_evidence_source_binding" "executive_approval" {
  environment            = kaleido_platform_environment.env_0.id
  service                = kaleido_platform_service.pms_0.id
  policy                 = kaleido_platform_pms_policy.tiered_approval.id
  policy_evidence_source = "executiveApproval"
  evidence_source_id     = kaleido_platform_pms_evidence_source.transfer_approval.id
  attesters              = "treasuryExecutives"
}

# A service request source acts as an application
resource "kaleido_platform_pms_policy_evidence_source_binding" "wallet_mapping" {
  environment            = kaleido_platform_environment.env_0.id
  service                = kaleido_platform_service.pms_0.id
  policy                 = kaleido_platform_pms_policy.tiered_approval.id
  policy_evidence_source = "walletMapping"
  evidence_source_id     = kaleido_platform_pms_evidence_source.wallet_lookup.id
  run_as                 = "ap:294hqr959b"
}

# A slot with no source: its evidence is seeded by a matcher or attached, and the
# mappings select the payload and attestation out of the message that arrives
resource "kaleido_platform_pms_policy_evidence_source_binding" "request" {
  environment            = kaleido_platform_environment.env_0.id
  service                = kaleido_platform_service.pms_0.id
  policy                 = kaleido_platform_pms_policy.tiered_approval.id
  policy_evidence_source = "request"
  payload_jsonata        = "body.request"
  attestation_jsonata    = "body.attestation"
}
