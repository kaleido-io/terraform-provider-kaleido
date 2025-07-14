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

resource "kaleido_platform_identity_provider" "kaleido_id" {
  name = "kaleido-id"
  hostname = "kaleido-id.kaleido.dev"
  client_type = "confidential"
  client_id = var.kaleido_id_client_id
  client_secret = var.kaleido_id_client_secret
  oidc_config_url = "https://sso-dev.photic.io/api/v1/.well-known/openid-configuration"
}

resource "kaleido_platform_account" "bootstrapped" {
  provider = kaleido.root

  name = "bootstrapped"
  oidc_client_id = kaleido_platform_identity_provider.kaleido_id.id
  hostnames = {
    "bootstrapped.kaleido.dev" = []
  }
  first_user_email = "hayden.fuss+buff@kaleido.io"
  user_jit_enabled = true
  user_jit_default_group = "readers"
  validation_policy = <<EOF
package sample_validation
import future.keywords.in

default allow := false

is_valid_aud(aud) := aud == ${var.kaleido_id_client_id}

is_valid_roles(roles) := roles == ["reader"] || roles == ["owner"] || roles == ["admin"]

allow {
	is_valid_aud(input.id_token.aud)
	is_valid_roles(input.id_token.roles)
}
EOF
  bootstrap_application_name = "kubernetes-local"
  bootstrap_application_oauth_json = jsonencode({
    issuer = var.bootstrap_application_issuer
    jwks_endpoint = var.bootstrap_application_jwks_endpoint
    ca_certificate = var.bootstrap_application_ca_certificate
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
