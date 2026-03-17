terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
    }
  }
}

provider "kaleido" {
  platform_api = var.kaleido_platform_api
  platform_username = var.kaleido_platform_username
  platform_password = var.kaleido_platform_password
}

resource "kaleido_platform_environment" "env_0" {
  name = var.environment_name
}

resource "kaleido_platform_runtime" "kmr_0" {
  type = "KeyManager"
  name = "kms1"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "kms_0" {
  type = "KeyManager"
  name = "kms1"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.kmr_0.id
  config_json = jsonencode({})
}
resource "kaleido_platform_kms_wallet" "wallet_0" {
  type = "azurekeyvault"
  name = "keystore1"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.kms_0.id
  creds_json = jsonencode({
    baseURL = var.azure_key_vault_base_url
    clientId = var.azure_app_registration_client_id
    clientSecret = var.azure_app_registration_client_secret
    tenantId = var.azure_app_registration_tenant_id
    keyVaultName = var.azure_key_vault_name
  })
  key_discovery_config = {
    secp256k1 = ["address_ethereum", "address_ethereum_checksum"]
  }
}