resource "kaleido_platform_pms_policy" "dual_approval" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  name        = "dual_approval"
  description = "Requires two approvals from the treasury operations list"

  # Bindings may be declared inline, in which case they are written in the same call
  # that creates the policy and its first version - so the definition below can already
  # reference them. Only the labels and names listed here are managed: a binding created
  # by a kaleido_platform_pms_policy_identity_list_binding,
  # kaleido_platform_pms_policy_evidence_source_binding or
  # kaleido_platform_pms_policy_output_formatter_binding resource, or by hand, is left
  # untouched.
  identity_list_binding = [
    {
      attester_label           = "treasuryOperations"
      identity_list_version_id = kaleido_platform_pms_identity_list.treasury_ops.applied_version_id
    }
  ]

  evidence_source_binding = [
    {
      policy_evidence_source = "approvers"
      evidence_source_id     = kaleido_platform_pms_evidence_source.transfer_approval.id
      attesters              = "treasuryOperations"
    }
  ]

  # The definition's output names this binding as its formatter
  output_formatter_binding = [
    {
      policy_output_formatter = "evmTransfer"
      output_formatter_id     = kaleido_platform_pms_output_formatter.evm_transfer.id
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
    # The bound formatter supplies the output type and the Rego that shapes the value;
    # the policy supplies a Rego expression for each of the formatter's parameters
    output = {
      formatter = "evmTransfer"
      values = {
        to     = { rego = "facts.to" }
        amount = { rego = "facts.amount" }
      }
    }
  })
}
