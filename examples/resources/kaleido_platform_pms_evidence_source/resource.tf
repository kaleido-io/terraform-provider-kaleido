# An evidence source is defined once, outside any policy, and referenced from a policy's
# evidence source bindings. Its JSONata evaluates against {request, decision}: 'request'
# is the object the policy's evidence 'request' block produced for the slot, and
# 'decision' carries {id, policy: {id, name, version}, evidence, idempotencyKey}.

# Evidence fetched from a platform service
resource "kaleido_platform_pms_evidence_source" "wallet_lookup" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  name        = "walletLookup"
  description = "Looks a wallet up by name or ID in the wallet manager"
  type        = "serviceRequest"

  request_schema_json = jsonencode({
    type       = "object"
    properties = { walletNameOrId = { type = "string" } }
    required   = ["walletNameOrId"]
  })

  service_request = {
    service      = "myWalletManager"
    type         = "WalletManagerService"
    options_json = jsonencode({ method = "GET", endpoint = "rest" })
    dynamic_options = {
      path_jsonata = "\"/wallets/\" & request.walletNameOrId"
    }
  }
}

# Evidence gathered by asking identities to approve or reject; who is asked is decided by
# the binding's 'attesters'
resource "kaleido_platform_pms_evidence_source" "transfer_approval" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  name        = "transferApproval"
  type        = "approval"

  request_schema_json = jsonencode({
    type = "object"
    properties = {
      asset  = { type = "string" }
      to     = { type = "string" }
      from   = { type = "string" }
      amount = { type = "integer" }
    }
    required = ["asset", "to", "from", "amount"]
  })

  approval = {
    approve = {
      primary_type = "Approval"
      types_json = jsonencode({
        Approval = [{ name = "transfer", type = "Transfer" }]
        Transfer = [
          { name = "asset", type = "string" },
          { name = "to", type = "string" },
          { name = "from", type = "string" },
          { name = "amount", type = "uint256" },
        ]
      })
      message_jsonata = "{\"transfer\": {\"asset\": request.asset, \"to\": request.to, \"from\": request.from, \"amount\": request.amount}}"
    }
    reject = {
      primary_type = "Rejection"
      types_json = jsonencode({
        Rejection = [{ name = "transfer", type = "Transfer" }]
        Transfer = [
          { name = "asset", type = "string" },
          { name = "to", type = "string" },
          { name = "from", type = "string" },
          { name = "amount", type = "uint256" },
        ]
      })
      message_jsonata = "{\"transfer\": {\"asset\": request.asset, \"to\": request.to, \"from\": request.from, \"amount\": request.amount}}"
    }
  }
}
