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

locals {
  kaleido_platform_domain = regex("https://[^.]+\\.(.+)", var.kaleido_platform_api)[0]
}

resource "kaleido_platform_identity_provider" "kaleido_id" {
  name = "kaleido-id"
  hostname = "kaleido-id"
  client_type = "confidential"
  client_id = var.idp_client_id
  client_secret = var.idp_client_secret
  oidc_config_url = var.idp_oidc_config_url
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
  first_user_email = var.first_user_email
  user_jit_enabled = true
  user_jit_default_group = "readers"
  validation_policy = <<EOF
package sample_validation
import future.keywords.in

default allow := false

is_valid_aud(aud) := aud == "${var.idp_client_id}"

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
    caCertificate = var.bootstrap_application_ca_certificate != "" ? var.bootstrap_application_ca_certificate : null
  })
  bootstrap_application_validation_policy = <<EOF
package k8s_application_validation

default allow := false

is_valid_sa(sub) := sub == "system:serviceaccount:${var.bootstrap_application_k8s_namespace}:${var.bootstrap_application_k8s_sa_name}"

allow {
  is_valid_sa(input.sub)
}
EOF

}

provider "kaleido" {
  platform_api = "https://bootstrapped.${local.kaleido_platform_domain}"
  platform_bearer_token = var.kaleido_platform_bearer_token

  alias = "bootstrapped"
}

resource "kaleido_platform_group" "readers" {
  provider = kaleido.bootstrapped

  name = "readers"
  depends_on = [kaleido_platform_account.bootstrapped]
}
