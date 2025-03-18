
variable "besu_testnet_name" {
    type = string
    default = "testnet"
}

variable "ipfs_testnet_name" {
    type = string
    default = "filenet"
}

variable "originator_api_url" {
  type = string
}

variable "originator_api_key_name" {
  type = string
}

variable "originator_api_key_value" {
  type = string
}

variable "joiner_one_api_url" {
  type = string
}

variable "joiner_one_api_key_name" {
  type = string
}

variable "joiner_one_api_key_value" {
  type = string
}

variable "joiner_two_api_url" {
  type = string
}

variable "joiner_two_api_key_name" {
  type = string
}

variable "joiner_two_api_key_value" {
  type = string
}

variable "originator_name" {
  type = string
}

variable "joiner_one_name" {
  type = string
}

variable "joiner_two_name" {
  type = string
}

variable "originator_signer_count" {
  type = number
  default = 1
}

variable "originator_peer_count" {
  type = number
  default = 1
}

variable "joiner_one_peer_count" {
  type = number
  default = 1
}

variable "joiner_two_peer_count" {
  type = number
  default = 1
}

variable "originator_peer_network_dz" {
  type = string
}

variable "joiner_one_peer_network_dz" {
  type = string
}

variable "joiner_two_peer_network_dz" {
  type = string
}

variable "originator_gateway_count" {
  type = number
  default = 1
  validation {
    condition = contains([0, 1], var.originator_gateway_count)
    error_message = "Valid values for originator_gateway_count are (0, 1)"
  }
}

variable "joiner_one_gateway_count" {
  type = number
  default = 1
  validation {
    condition = contains([0, 1], var.joiner_one_gateway_count)
    error_message = "Valid values for joiner_one_gateway_count are (0, 1)"
  }
}

variable "joiner_two_gateway_count" {
  type = number
  default = 1
  validation {
    condition = contains([0, 1], var.joiner_two_gateway_count)
    error_message = "Valid values for joiner_two_gateway_count are (0, 1)"
  }
}
