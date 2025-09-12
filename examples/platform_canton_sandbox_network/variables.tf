
variable "api_key_name" {
    type = string
}
variable "api_key_value" {
    type = string
}

variable "kaleido_platform_baseurl" {
    type = string
}

variable "environment_name" {
    type = string
}

variable "sv_node_size" {
  type = string
  default = "Large"
}

variable "participant_node_size" {
  type = string
  default = "Medium"
}

variable "canton_wallet_name" {
  type = string
  default = "canton-wallet"
}

variable "enable_synchronizer_network" {
  type = bool
  default = false
}

variable "synchronizer_node_size" {
  type = string
  default = "Small"
}

variable "canton_key_spec" {
  type = string
  default = "secp256r1"
}