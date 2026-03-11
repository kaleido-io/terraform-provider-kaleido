variable "kaleido_platform_api" {
  type        = string
  description = "Kaleido platform API URL"
}

variable "api_key_name" {
  type        = string
  description = "API key name used to authenticate with the platform"
}

variable "api_key_value" {
  type        = string
  sensitive   = true
  description = "API key secret used to authenticate with the platform"
}

variable "application_name" {
  type        = string
  default     = "env-scoped-app"
  description = "Name of the application to create"
}

variable "new_api_key_name" {
  type        = string
  default     = "env-scoped-api-key"
  description = "Name of the API key to generate for the application"
}

variable "rego_policy" {
  type        = string
  description = "Rego policy document to attach to the application. If empty, loads policy.rego from the module directory."
  default     = ""
}
