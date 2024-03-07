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

variable "node_count" {
  type = number
}

variable "json_rpc_url" {
  type = string
}

variable "json_rpc_username" {
  type = string
  default = ""
}

variable "json_rpc_password" {
  type = string
  default = ""
}