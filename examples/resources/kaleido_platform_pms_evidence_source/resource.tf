# An evidence source is defined once, outside any policy, and referenced from a policy's
# evidence source bindings. It declares the parameters a policy must supply in its
# evidence 'request' block, and the JSON Schema of one item of the evidence it produces,
# which the policy composer reads for the slots bound to it. Its JSONata evaluates against
# {request, decision, body}: 'request' is the object the policy's evidence 'request' block
# produced for the slot, 'decision' carries {id, policy: {id, name, version}, evidence,
# idempotencyKey}, and 'body' is what the source received.

# Evidence fetched from a platform service
resource "kaleido_platform_pms_evidence_source" "wallet_lookup" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  name        = "walletLookup"
  description = "Looks a wallet up by name or ID in the wallet manager"
  type        = "serviceRequest"

  parameter = [
    {
      name        = "walletNameOrId"
      type        = "string"
      description = "The wallet to look up"
    }
  ]

  schema_json = jsonencode({
    type = "object"
    properties = {
      id   = { type = "string" }
      name = { type = "string" }
    }
  })

  # Select the evidence out of the service response
  payload_jsonata = "body"

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
# the binding's 'attesters'. The schema of an approval source is derived by the server
# from the typed data of its responses, so schema_json is not set here.
resource "kaleido_platform_pms_evidence_source" "transfer_approval" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  name        = "transferApproval"
  type        = "approval"

  parameter = [
    { name = "asset", type = "string" },
    { name = "to", type = "string" },
    { name = "from", type = "string" },
    { name = "amount", type = "number" },
  ]

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

# Evidence that is pushed in rather than requested: seeded by a matcher, attached manually
# or supplied late-bound. An attachment source only describes its shape and how to select
# the payload and attestation out of what arrives.
resource "kaleido_platform_pms_evidence_source" "signed_document" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.pms_0.id
  name        = "signedDocument"
  type        = "attachment"

  schema_json = jsonencode({
    type = "object"
    properties = {
      document = { type = "string" }
    }
    required = ["document"]
  })

  payload_jsonata     = "body.document"
  attestation_jsonata = "body.signature"
}
