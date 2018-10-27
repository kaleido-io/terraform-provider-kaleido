variable "kaleido_api_key" {
  type = "string"
  description = "Kaleido API Key"
}

variable "kaleido_region" {
  type = "string"
  description = "Can be '-ap' for Sydney, or '-eu' for Frankfurt. Defaults to US"
  default = ""
}

variable "env_types" {
  type = "list"
  default = ["quorum", "geth"]
  description = "List of environment types you want to deploy. Options are 'quorum' and 'geth'."
}

variable "quorum_consensus" {
  type = "list"
  default = ["raft", "ibft"]
  description = "Consensus methods supported by quorum."
}