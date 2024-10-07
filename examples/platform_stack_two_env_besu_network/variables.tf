
variable "originator_api_url" {
  type = string
}

variable "originator_api_key_name" {
  type = string
}

variable "originator_api_key_value" {
  type = string
}

variable "secondary_api_url" {
  type = string
}

variable "secondary_api_key_name" {
  type = string
}

variable "secondary_api_key_value" {
  type = string
}

variable "originator_name" {
  type = string
}

variable "secondary_name" {
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

variable "secondary_peer_count" {
  type = number
  default = 1
}

variable "originator_peer_network_dz" {
  type = string
}

variable "originator_peer_network_connection" {
  type = string
}

variable "secondary_peer_network_dz" {
  type = string
}

variable "secondary_peer_network_connection" {
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

variable "secondary_gateway_count" {
  type = number
  default = 1
  validation {
    condition = contains([0, 1], var.secondary_gateway_count)
    error_message = "Valid values for secondary_gateway_count are (0, 1)"
  }
}
