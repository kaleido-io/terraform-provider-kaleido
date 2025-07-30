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
  platform_bearer_token = var.kaleido_platform_bearer_token
}

# Create development account
resource "kaleido_platform_account" "dev_account" {
  name                = var.dev_account_name
  oidc_client_id      = var.oidc_client_id
  validation_policy   = var.validation_policy
  first_user_email    = var.dev_admin_email
  
  hostnames = {
    "${var.dev_hostname}" = []
  }
}

# Create staging account
resource "kaleido_platform_account" "staging_account" {
  name                = var.staging_account_name
  oidc_client_id      = var.oidc_client_id
  validation_policy   = var.validation_policy
  first_user_email    = var.staging_admin_email
  
  hostnames = {
    "${var.staging_hostname}" = []
  }
}

# Create production account
resource "kaleido_platform_account" "production_account" {
  name                = var.production_account_name
  oidc_client_id      = var.oidc_client_id
  validation_policy   = var.validation_policy
  first_user_email    = var.production_admin_email
  
  hostnames = {
    "${var.production_hostname}" = []
  }
} 