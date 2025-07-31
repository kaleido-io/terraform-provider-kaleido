terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
    }
  }
}

provider "kaleido" {
  platform_api = var.kaleido_platform_api
  platform_bearer_token = var.kaleido_platform_bearer_token

  alias = "root"
}

variable "kaleido_platform_api" {
  type = string
  default = "https://account1.kaleido.dev"
}

variable "kaleido_platform_bearer_token" {
  type = string
}

variable "bootstrap_application_issuer" {
  type = string
  default = "https://kubernetes.default.svc.cluster.local"
}

variable "bootstrap_application_jwks_endpoint" {
  type = string
  default = "https://kubernetes.default.svc.cluster.local/openid/v1/jwks"
}

variable "bootstrap_application_ca_certificate" {
  type = string
}

variable "kaleido_id_client_id" {
  type = string
}

variable "kaleido_id_client_secret" {
  type = string
}

variable "kaleido_id_config_url" {
  type = string
}

variable "kaleido_id_first_user_email" {
  type = string
}

resource "kaleido_platform_identity_provider" "kaleido_id" {
  name = "kaleido-id"
  hostname = "kaleido-id"
  client_type = "confidential"
  client_id = var.kaleido_id_client_id
  client_secret = var.kaleido_id_client_secret
  oidc_config_url = var.kaleido_id_config_url
  id_token_nonce_enabled = true

  provider = kaleido.root
}

resource "kaleido_platform_account" "bootstrapped" {
  provider = kaleido.root

  name = "bootstrapped"
  oidc_client_id = kaleido_platform_identity_provider.kaleido_id.id
  hostnames = {
    "bootstrapped" = []
  }
  first_user_email = var.kaleido_id_first_user_email
  user_jit_enabled = true
  user_jit_default_group = "readers"
  validation_policy = <<EOF
package sample_validation
import future.keywords.in

default allow := false

is_valid_aud(aud) := aud == "${var.kaleido_id_client_id}"

is_valid_reader(roles) := roles == ["reader"]

is_valid_admin(roles) := roles == ["admin"]

is_valid_owner(roles) := roles == ["owner"]

allow {
	is_valid_aud(input.id_token.aud)
	is_valid_reader(input.id_token.roles)
}

allow {
	is_valid_aud(input.id_token.aud)
	is_valid_admin(input.id_token.roles)
}

allow {
	is_valid_aud(input.id_token.aud)
	is_valid_owner(input.id_token.roles)
}
EOF
  bootstrap_application_name = "kubernetes-local"
  bootstrap_application_oauth_json = jsonencode({
    issuer = var.bootstrap_application_issuer
    jwksEndpoint = var.bootstrap_application_jwks_endpoint
    caCertificate = var.bootstrap_application_ca_certificate
  })
  bootstrap_application_validation_policy = <<EOF
package k8s_application_validation

default allow := false

is_valid_sa(sub) := sub == "system:serviceaccount:default:kaleidoplatform"

allow {
  is_valid_sa(input.sub)
}
EOF

}

provider "kaleido" {
  platform_api = "https://bootstrapped.kaleido.dev"
  platform_bearer_token = var.kaleido_platform_bearer_token

  alias = "bootstrapped"
}

resource "kaleido_platform_group" "readers" {
  provider = kaleido.bootstrapped

  name = "readers"
  depends_on = [kaleido_platform_account.bootstrapped]
}
