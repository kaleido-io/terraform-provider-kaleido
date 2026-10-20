resource "kaleido_platform_pms_policy_matcher" "bond_transfers" {
  environment       = kaleido_platform_environment.env_0.id
  service           = kaleido_platform_service.pms_0.id
  policy            = kaleido_platform_pms_policy.dual_approval.id
  enforcement_point = "wfe-hook"

  # Apply the policy only to workflow transactions labelled as bond transfers
  match_json = jsonencode({
    equal = [
      {
        field = "label.assetType"
        value = "bond"
      }
    ]
  })

  parameters_json = jsonencode({
    minApprovals = 3
  })

  # Seed the "documents" evidence slot from the transaction input
  evidence = [
    {
      slot                = "documents"
      payload_jsonata     = "$.input.document"
      attestation_jsonata = "$.input.signature"
    }
  ]
}
