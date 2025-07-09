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

# Create a shared OIDC identity provider
resource "kaleido_platform_identity_provider" "shared_oidc" {
  name         = var.identity_provider_name
  client_id    = var.oidc_client_id
  client_secret = var.oidc_client_secret
  client_type  = "confidential"
  hostname     = var.hostname
  
  # OIDC configuration - either provide oidc_config_url or individual endpoints
  oidc_config_url = var.oidc_config_url
  
  # Individual endpoint configuration (used if oidc_config_url not provided)
  issuer       = var.issuer
  login_url    = var.login_url
  token_url    = var.token_url
  logout_url   = var.logout_url
  jwks_url     = var.jwks_url
  user_info_url = var.user_info_url
  
  # Security settings
  scopes = var.scopes
  confidential_pkce_enabled = true
  id_token_nonce_enabled    = true
}

# Create development account
resource "kaleido_platform_account" "dev_account" {
  name                = var.dev_account_name
  oidc_client_id      = kaleido_platform_identity_provider.shared_oidc.id
  validation_policy   = var.validation_policy
  first_user_email    = var.dev_admin_email
  first_user_sub      = var.dev_admin_sub
  
  hostnames = var.dev_hostnames
}

# Create staging account
resource "kaleido_platform_account" "staging_account" {
  name                = var.staging_account_name
  oidc_client_id      = kaleido_platform_identity_provider.shared_oidc.id
  validation_policy   = var.validation_policy
  first_user_email    = var.staging_admin_email
  first_user_sub      = var.staging_admin_sub
  
  hostnames = var.staging_hostnames
}

# Create production account
resource "kaleido_platform_account" "production_account" {
  name                = var.production_account_name
  oidc_client_id      = kaleido_platform_identity_provider.shared_oidc.id
  validation_policy   = var.validation_policy
  first_user_email    = var.production_admin_email
  first_user_sub      = var.production_admin_sub
  
  hostnames = var.production_hostnames
} 