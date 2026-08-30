resource "kaleido_platform_pms_policy" "dual_approval" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  name        = "dual_approval"
  description = "Requires two approvals from the treasury operations list"

  # Bindings may be declared inline, in which case they are written in the same call
  # that creates the policy and its first version - so the definition below can already
  # reference them. Only the labels and names listed here are managed: a binding created
  # by a kaleido_platform_pms_policy_identity_list_binding or
  # kaleido_platform_pms_policy_evidence_source_binding resource, or by hand, is left
  # untouched.
  identity_list_binding = [
    {
      attester_label           = "treasuryOperations"
      identity_list_version_id = kaleido_platform_pms_identity_list.treasury_ops.applied_version_id
    }
  ]

  evidence_source_binding = [
    {
      name = "approvers"
      type = "approval"
      approval = {
        approval = {
          payload_type             = "TypedDataV4"
          payload_template_jsonata = "$.request"
        }
        identity_list_version = {
          id = kaleido_platform_pms_identity_list.treasury_ops.applied_version_id
        }
      }
    }
  ]

  definition_yaml = yamlencode({
    parameters = [
      {
        name        = "minApprovals"
        type        = "number"
        default     = 2
        description = "How many approvals are required"
      }
    ]
    evidence = [
      {
        name   = "approvals"
        source = "approvers"
        attestation = {
          type      = "eip712"
          attesters = "treasuryOperations"
        }
      }
    ]
    decision = {
      gate = {
        evidence = ["approvals"]
      }
    }
    output = {
      type = "kaleido.policy.evm.v1"
    }
  })
}
