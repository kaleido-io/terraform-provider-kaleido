# An output formatter is defined once, outside any policy, and referenced from a policy's
# output formatter bindings. It declares the envelope type consumers discriminate on, the
# parameters a policy must feed it, and the Rego mapping that shapes those parameters into
# the output value. The formatter is pinned to each policy version when the version is
# created, so changing it here does not alter versions that already exist.
resource "kaleido_platform_pms_output_formatter" "evm_transfer" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  name        = "evmTransfer"
  description = "Shapes an approved transfer for submission to an EVM chain"
  type        = "kaleido.policy.evm.v1"

  parameter = [
    {
      name        = "to"
      type        = "string"
      description = "The recipient address"
    },
    {
      name         = "amount"
      type         = "number"
      description  = "The amount to transfer"
      default_json = jsonencode(0)
    },
  ]

  # A single Rego expression over parameters.<name>
  mapping_rego = "{\"to\": parameters.to, \"value\": parameters.amount}"
}
