variable "kaleido_api_key" {
  type = string
  description = "Kaleido API Key"
}

variable "kaleido_region" {
  type = string
  description = "Can be '-ap' for Sydney, or '-eu' for Frankfurt. Defaults to US"
  default = ""
}

variable "provider_type" {
  type = string
  default = "pantheon"
  description = "Protocol implementation to deploy."
}

variable "consensus" {
  type = string
  default = "ibft"
  description = "Consensus mechanism."
}

variable "multi_region" {
  type = string
  default = true
  description = "Make the environment multi-region compatible to support additional regions, now or in the future"
}

variable "node_size" {
  type = string
  default = "small"
  description = "Size of the node"
}

variable "node_count" {
  type = string
  default = 4
  description = "Count of nodes to create - each will have its own membership"
}

variable "service_count" {
  type = string
  default = 1
  description = "Count of services to create - each will have its own membership"
}

variable "consortium_name" {
  type = string
  default = "My Business Network"
}

variable "env_name" {
  type = string
  default = "Development"
}

variable "env_description" {
  type = string
  default = "Created with Terraform"
}
variable "network_description" {
  type = string
  default = "Modern Business Network - Built on Kaleido"
}
