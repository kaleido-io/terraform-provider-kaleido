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

variable "signer_node_count" {
  type = number
  default=1
}

variable "nonsigner_node_count" {
  type = number
  default=1
}
