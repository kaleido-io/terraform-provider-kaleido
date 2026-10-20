# A binding gives the policy definition's 'output.formatter' a name that resolves to an
# output formatter, so the formatter can be swapped without editing the definition.
resource "kaleido_platform_pms_policy_output_formatter_binding" "evm_transfer" {
  environment             = kaleido_platform_environment.env_0.id
  service                 = kaleido_platform_service.pms_0.id
  policy                  = kaleido_platform_pms_policy.tiered_approval.id
  policy_output_formatter = "evmTransfer"
  output_formatter_id     = kaleido_platform_pms_output_formatter.evm_transfer.id
}
