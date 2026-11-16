resource "kaleido_platform_group" "treasury" {
  name = "treasury"
}

# The treasury group signs with the treasury keys in the payments keystore.
resource "kaleido_platform_grant" "treasury_signers" {
  name              = "treasury-signers-payments"
  group_id          = kaleido_platform_group.treasury.id
  permission_set_id = kaleido_platform_permission_set.key_signers.id
  params_json       = jsonencode({ keystore = "payments" })
}

# ...and can reach the Key Manager's endpoint, with the built-in service-api-user set.
data "kaleido_platform_permission_set" "service_api_user" {
  name = "service-api-user"
}

resource "kaleido_platform_grant" "treasury_km_host" {
  name              = "treasury-km1-host"
  group_id          = kaleido_platform_group.treasury.id
  permission_set_id = data.kaleido_platform_permission_set.service_api_user.id
  params_json       = jsonencode({ service = kaleido_platform_service.km1.id })
}

# A grant with an inline set of its own: read every key in the Key Manager.
resource "kaleido_platform_grant" "auditor_keys" {
  name           = "auditor-km1-keys"
  application_id = kaleido_platform_application.auditor.id
  permission_set_json = jsonencode({
    permissions = [{
      actions = ["read"]
      resourceSelector = [
        { field = "resource.type", op = "eq", values = ["keymanager.key"] },
        { field = "resource.within.service.name", op = "eq", values = ["km1"] },
      ]
    }]
  })
}
