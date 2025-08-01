# ============================================================================
# Root Account & Authentication Variables
# ============================================================================

variable "root_platform_api" {
  description = "Root account platform API URL (e.g., https://account1.kaleido.dev)"
  type        = string
  default     = "https://account1.kaleido.dev"
}

variable "root_platform_username" {
  description = "Root account platform username"
  type        = string
}

variable "root_platform_password" {
  description = "Root account platform password"
  type        = string
  sensitive   = true
}

variable "bootstrap_platform_bearer_token" {
  description = "Bearer token for root account authentication (from Kubernetes service account)"
  type        = string
  sensitive   = true
}

# ============================================================================
# OAuth & Identity Provider Configuration
# ============================================================================

variable "kaleido_id_client_id" {
  description = "Kaleido ID OIDC client ID"
  type        = string
}

variable "kaleido_id_client_secret" {
  description = "Kaleido ID OIDC client secret"
  type        = string
  sensitive   = true
}

variable "kaleido_id_config_url" {
  description = "Kaleido ID OIDC configuration URL"
  type        = string
}

variable "first_user_email" {
  description = "Email address for the first user in each child account"
  type        = string
}

# ============================================================================
# Bootstrap Application Configuration (for Kubernetes service account trust)
# ============================================================================

variable "bootstrap_application_issuer" {
  description = "Bootstrap application issuer URL (typically Kubernetes cluster)"
  type        = string
  default     = "https://kubernetes.default.svc.cluster.local"
}

variable "bootstrap_application_jwks_endpoint" {
  description = "Bootstrap application JWKS endpoint URL"
  type        = string
  default     = "https://kubernetes.default.svc.cluster.local/openid/v1/jwks"
}

variable "bootstrap_application_ca_certificate" {
  description = "Bootstrap application CA certificate for Kubernetes cluster"
  type        = string
}

# ============================================================================
# Child Account Configuration
# ============================================================================

variable "child_account_count" {
  description = "Total number of child accounts to create (2-5 accounts supported in this example)"
  type        = number
  default     = 3
  
  validation {
    condition     = var.child_account_count >= 2 && var.child_account_count <= 5
    error_message = "Child account count must be between 2 and 5 for this example. For more accounts, extend the pattern or use multiple deployments."
  }
}

variable "account_name_prefix" {
  description = "Prefix for child account names (e.g., 'node' creates 'node1', 'node2', etc.)"
  type        = string
  default     = "node"
}

# ============================================================================
# Network Configuration
# ============================================================================

variable "besu_network_name" {
  description = "Base name for Besu networks"
  type        = string
  default     = "testnet"
}

variable "ipfs_network_name" {
  description = "Base name for IPFS networks"
  type        = string
  default     = "filenet"
}

variable "paladin_network_name" {
  description = "Base name for Paladin networks"
  type        = string
  default     = "paladinnet"
}

variable "chain_id" {
  description = "Chain ID for the Besu network"
  type        = number
  default     = 12345
}

variable "block_period_seconds" {
  description = "Block period in seconds for QBFT consensus"
  type        = number
  default     = 5
}

variable "initial_balances" {
  description = "Initial account balances for the Besu network"
  type        = map(string)
  default     = {}
  # default = {
  #   "0x12F62772C4652280d06E64CfBC9033d409559aD4" = "0x111111111111"
  # }
}

# ============================================================================
# Infrastructure Configuration
# ============================================================================

variable "deployment_zone" {
  description = "Deployment zone for peerable nodes (e.g., 'platform-conn')"
  type        = string
  default     = "platform-conn"
}

variable "originator_signer_count" {
  description = "Number of signer (validator) nodes in the originator account"
  type        = number
  default     = 1
  
  validation {
    condition     = var.originator_signer_count >= 1
    error_message = "At least 1 signer node is required for consensus."
  }
}

variable "gateway_count" {
  description = "Number of EVM gateways per account (0 or 1)"
  type        = number
  default     = 1
  
  validation {
    condition     = contains([0, 1], var.gateway_count)
    error_message = "Gateway count must be 0 or 1."
  }
}

# ============================================================================
# Connectivity Configuration
# ============================================================================

# Note: This module implements full mesh connectivity only
# All Paladin nodes require P2P networking to all fellow nodes for selective disclosure use cases

# ============================================================================
# Optional Advanced Configuration
# ============================================================================

variable "paladin_node_size" {
  description = "Size configuration for Paladin nodes"
  type        = string
  default     = "Small"
  
  validation {
    condition = contains(["Small", "Medium", "Large"], var.paladin_node_size)
    error_message = "Paladin node size must be Small, Medium, or Large."
  }
}

variable "custom_validation_policy" {
  description = "Custom validation policy for child accounts (if null, uses default)"
  type        = string
  default     = null
}

variable "enable_force_delete" {
  description = "Enable force delete for Besu services (useful for development)"
  type        = bool
  default     = true
}

