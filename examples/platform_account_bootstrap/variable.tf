variable "kaleido_platform_api" {
  type = string
  default = "https://account1.kaleido.dev"
  description = "Set the Kaleido Platform API URL to your account1 endpoint to create the child accounts in this module"
}

variable "kaleido_platform_bearer_token" {
  type = string
}

variable "idp_client_id" {
  type = string
  description = "Set the client ID for the new account's identity provider"
}

variable "idp_client_secret" {
  type = string
  description = "Set the client secret for the new account's identity provider"
}

variable "idp_oidc_config_url" {
  type = string
  description = "Set the OIDC config URL for new account's identity provider"
}

variable "first_user_email" {
  type = string
  description = "Set the email address of the initial admin user for the new account"
}
variable "bootstrap_application_issuer" {
  type = string
  default = "https://kubernetes.default.svc.cluster.local"
  description = "Set the issuer URL for the bootstrap OAuth application"
}

variable "bootstrap_application_jwks_endpoint" {
  type = string
  default = "https://kubernetes.default.svc.cluster.local/openid/v1/jwks"
  description = "Set the JWKS endpoint for the bootstrap OAuth application"
}

variable "bootstrap_application_ca_certificate" {
  type = string
  default = ""
  description = "Set the CA certificate for the bootstrap OAuth application"
}

variable "bootstrap_application_k8s_namespace" {
  type = string
  default = "default"
  description = "Set the Kubernetes namespace where your service account exists"
}


variable "bootstrap_application_k8s_sa_name" {
  type = string
  default = "kaleidoplatform"
  description = "Set the Kubernetes service account name you are using to authenticate with the Kaleido Platform."
}