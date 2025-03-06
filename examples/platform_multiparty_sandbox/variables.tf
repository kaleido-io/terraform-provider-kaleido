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

variable "besu_node_count" {
  type = number
  default = 1
}

variable "members" {
  type = list(string)
}

variable "pdm_manage_p2p_tls" {
    type = bool
    default = false
}

variable "pdm_runtime_endpoint_domain" {
  type = string
  default = "use2.kaleido.local" // contact support for obtaining this value for your platform instance
}

variable "pdm_service_peer_id" {
    type = string
    default = "testnet-pdm"
}
