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
}

variable "rpc_url" {
  type = string
  description = "The RPC endpoint for the public chain"
}

variable "username" {
  type = string
  description = "The username for connecting to the public chain"
}

variable "password" {
  type = string
  description = "The password for connecting to the public chain"
}
