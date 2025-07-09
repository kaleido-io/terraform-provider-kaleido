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

variable "oidc_client_id" {
  description = "OIDC client ID"
  type        = string
}

variable "validation_policy" {
  description = "Account validation policy"
  type        = string
  default     = null
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

variable "dev_hostname" {
  description = "Hostname mappings for development account"
  type        = string
  default = "dev"
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

variable "staging_hostname" {
  description = "Hostname mappings for staging account"
  type        = string
  default = "stage"
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

variable "production_hostname" {
  description = "Hostname mappings for production account"
  type        = string
  default = "prod"
} 