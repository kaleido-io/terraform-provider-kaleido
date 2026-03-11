terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
    }
  }
}

provider "kaleido" {
  platform_api      = var.kaleido_platform_api
  platform_username = var.api_key_name
  platform_password = var.api_key_value
}

resource "kaleido_platform_application" "app" {
  name          = var.application_name
  admin_enabled = false
  oauth_enabled = false
}

resource "kaleido_platform_account_access_policy" "policy" {
  application_id = kaleido_platform_application.app.id
  policy         = var.rego_policy != "" ? var.rego_policy : file("${path.module}/policy.rego")
}

resource "kaleido_platform_api_key" "app_key" {
  name           = var.new_api_key_name
  application_id = kaleido_platform_application.app.id
  no_expiry      = true
}

output "api_key_secret" {
  value     = kaleido_platform_api_key.app_key.secret
  sensitive = true
}
