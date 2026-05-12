
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
    default = "raas"
}

// L2 settings
variable "l2_network_name" {
  type = string
  default = "l2"
}

variable "l2_node_count" {
  type = number
  default = 1
}

variable "l2_node_size" {
  type = string
  default = "Medium"
}

variable "l2_rollup_file" {
  type = string
  default = "resources/l2/rollup.json"
}

variable "l2_genesis_file" {
  type = string
  default = "resources/l2/genesis.json"
}

variable "l2_sequencer_key" {
  type = string
}

variable "l2_batcher_key" {
  type = string
}

variable "l2_proposer_key" {
  type = string
}

variable "l2_namespace_file" {
  type = string
  default = "resources/l2/namespace"
}

// L1 settings
variable "l1_chain_id" {
  type = number
  default = 12345
}

variable "l1_endpoint" {
  type = string
}

variable "l1_game_factory_address" {
  type = string
}

// DA Network settings
variable "da_type" {
  type = string
  default = "OnChain"
}

variable "da_network_name" {
  type = string
  default = "da-network"
}

variable "da_network_type" {
  type = string
  description = "The type of network to create, Public or Custom"
  default = "Custom"
}

variable "da_chain_id" {
  type = string
  description = "The chain ID of the Celestia network"
}

variable "da_node_type" {
  type = string
  description = "The type of node to create"
  default = "bridge"
}

variable "da_node_size" {
  type = string
  default = "Medium"
}

variable "da_init_mode" {
  type = string
  default = "manual"
}

variable "da_genesis_file"{
  type = string
  default = "resources/da/genesis.json"
}

variable "da_validator_seeds" {
  type = list(string)
}

variable "funded_account_seeds" {
  type = list(string)
}

variable "funded_account_addresses" {
  type = list(string)
}



