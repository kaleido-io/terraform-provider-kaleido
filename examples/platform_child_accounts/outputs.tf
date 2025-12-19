
output "dev_account" {
  description = "Development account details"
  value = {
    id                = kaleido_platform_account.dev_account.id
    name              = kaleido_platform_account.dev_account.name
    oidc_client_id    = kaleido_platform_account.dev_account.oidc_client_id
    validation_policy = kaleido_platform_account.dev_account.validation_policy
    hostnames         = kaleido_platform_account.dev_account.hostnames
  }
}

output "staging_account" {
  description = "Staging account details"
  value = {
    id                = kaleido_platform_account.staging_account.id
    name              = kaleido_platform_account.staging_account.name
    oidc_client_id    = kaleido_platform_account.staging_account.oidc_client_id
    validation_policy = kaleido_platform_account.staging_account.validation_policy
    hostnames         = kaleido_platform_account.staging_account.hostnames
  }
}

output "production_account" {
  description = "Production account details"
  value = {
    id                = kaleido_platform_account.production_account.id
    name              = kaleido_platform_account.production_account.name
    oidc_client_id    = kaleido_platform_account.production_account.oidc_client_id
    validation_policy = kaleido_platform_account.production_account.validation_policy
    hostnames         = kaleido_platform_account.production_account.hostnames
  }
}

output "all_accounts" {
  description = "All created accounts"
  value = {
    development = {
      id                = kaleido_platform_account.dev_account.id
      name              = kaleido_platform_account.dev_account.name
      oidc_client_id    = kaleido_platform_account.dev_account.oidc_client_id
      validation_policy = kaleido_platform_account.dev_account.validation_policy
      hostnames         = kaleido_platform_account.dev_account.hostnames
    }
    staging = {
      id                = kaleido_platform_account.staging_account.id
      name              = kaleido_platform_account.staging_account.name
      oidc_client_id    = kaleido_platform_account.staging_account.oidc_client_id
      validation_policy = kaleido_platform_account.staging_account.validation_policy
      hostnames         = kaleido_platform_account.staging_account.hostnames
    }
    production = {
      id                = kaleido_platform_account.production_account.id
      name              = kaleido_platform_account.production_account.name
      oidc_client_id    = kaleido_platform_account.production_account.oidc_client_id
      validation_policy = kaleido_platform_account.production_account.validation_policy
      hostnames         = kaleido_platform_account.production_account.hostnames
    }
  }
}

output "setup_summary" {
  description = "Summary of the platform setup"
  value = {
    accounts_created     = 3
    account_names        = [
      kaleido_platform_account.dev_account.name,
      kaleido_platform_account.staging_account.name,
      kaleido_platform_account.production_account.name
    ]
  }
} 