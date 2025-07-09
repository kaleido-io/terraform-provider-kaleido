variable "kaleido_platform_api" {
  description = "Kaleido platform API URL"
  type        = string
}

variable "kaleido_platform_username" {
  description = "Kaleido platform username"
  type        = string
}

variable "kaleido_platform_password" {
  description = "Kaleido platform password"
  type        = string
  sensitive   = true
}

variable "identity_provider_name" {
  description = "Name of the OIDC identity provider"
  type        = string
  default     = "shared-oidc-provider"
}

variable "oidc_client_id" {
  description = "OIDC client ID"
  type        = string
}

variable "oidc_client_secret" {
  description = "OIDC client secret"
  type        = string
  sensitive   = true
}

variable "hostname" {
  description = "Hostname for OIDC redirect URLs"
  type        = string
}

variable "oidc_config_url" {
  description = "OIDC configuration URL (e.g., https://example.com/.well-known/openid-configuration)"
  type        = string
  default     = null
}

variable "issuer" {
  description = "OIDC issuer URL (required if oidc_config_url not provided)"
  type        = string
  default     = null
}

variable "login_url" {
  description = "OIDC login URL (required if oidc_config_url not provided)"
  type        = string
  default     = null
}

variable "token_url" {
  description = "OIDC token URL (required if oidc_config_url not provided)"
  type        = string
  default     = null
}

variable "logout_url" {
  description = "OIDC logout URL"
  type        = string
  default     = null
}

variable "jwks_url" {
  description = "OIDC JWKS URL"
  type        = string
  default     = null
}

variable "user_info_url" {
  description = "OIDC user info URL"
  type        = string
  default     = null
}

variable "scopes" {
  description = "OIDC scopes to request"
  type        = string
  default     = "openid email profile"
}

variable "validation_policy" {
  description = "Account validation policy"
  type        = string
  default     = "strict"
}

# Development account variables
variable "dev_account_name" {
  description = "Name of the development account"
  type        = string
  default     = "development"
}

variable "dev_admin_email" {
  description = "Email of the development account admin"
  type        = string
}

variable "dev_admin_sub" {
  description = "OIDC subject identifier for the development account admin"
  type        = string
}

variable "dev_hostnames" {
  description = "Hostname mappings for development account"
  type        = map(list(string))
  default = {
    "dev.example.com" = ["rest", "websocket"]
  }
}

# Staging account variables
variable "staging_account_name" {
  description = "Name of the staging account"
  type        = string
  default     = "staging"
}

variable "staging_admin_email" {
  description = "Email of the staging account admin"
  type        = string
}

variable "staging_admin_sub" {
  description = "OIDC subject identifier for the staging account admin"
  type        = string
}

variable "staging_hostnames" {
  description = "Hostname mappings for staging account"
  type        = map(list(string))
  default = {
    "staging.example.com" = ["rest"]
  }
}

# Production account variables
variable "production_account_name" {
  description = "Name of the production account"
  type        = string
  default     = "production"
}

variable "production_admin_email" {
  description = "Email of the production account admin"
  type        = string
}

variable "production_admin_sub" {
  description = "OIDC subject identifier for the production account admin"
  type        = string
}

variable "production_hostnames" {
  description = "Hostname mappings for production account"
  type        = map(list(string))
  default = {
    "prod.example.com" = ["rest", "websocket"]
  }
} 