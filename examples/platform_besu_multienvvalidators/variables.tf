variable "kaleido_platform_api" {
  type = string
  default = "https://account1.kaleido.dev"
}

variable "kaleido_platform_api_key_name" {
  type = string
}

variable "kaleido_platform_api_key_secret" {
  type = string
}

variable "node_peerable_zone" {
  type = string
  default = "platform-conn-zone-1"
}

variable "network_name" {
  type = string
  default = "testnet"
}

variable "validators" {
  type = map(object({
    name = string
  }))
  default = {
    "validator1" = {
      name = "validator1"
    }
    "validator2" = {
      name = "validator2"
    }
    "validator3" = {
      name = "validator3"
    }
    "validator4" = {
      name = "validator4"
    }
  }
}

variable "block_period" {
  type = number
  default = 3
}

variable "chain_id" {
  type = number
  default = 3333
}
