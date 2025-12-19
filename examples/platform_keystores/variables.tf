variable "kaleido_platform_api" {
  type = string
}

variable "kaleido_platform_username" {
  type = string
}

variable "kaleido_platform_password" {
  type = string
}

variable "environment_name" {
  type = string
  default = "keystores-example"
}

variable "azure_key_vault_name" {
  type = string
  description = "The Name of the Azure Key Vault where you want to fetch keys"
}

variable "azure_key_vault_base_url" {
  type = string
  description = "The base URL of the Azure Key Vault where you want to fetch your keys"
 default = "https://vault.azure.net"
}

variable "azure_app_registration_client_id" {
  type = string
  description = "The client ID for the Azure App Registration"
}

variable "azure_app_registration_client_secret" {  
  type = string
  description = "The client secret of the Azure App Registration"
}

variable "azure_app_registration_tenant_id" {
  type = string
  description = "The tenant ID of the Azure App Registration"
}

