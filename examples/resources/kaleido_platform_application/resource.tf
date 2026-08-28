// Simple API key application
resource "kaleido_platform_application" "application" {
  name          = "application"
  admin_enabled = true
  oauth_enabled = false
}

// OAuth application with static JWKS (third-party providers are used illustratively)
terraform {
  required_providers {
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
    jwks = {
      source  = "iwarapter/jwks"
      version = "~> 0.0"
    }
  }
}

provider "tls" {}
provider "jwks" {}

# 1. Generate the RSA Private Key
resource "tls_private_key" "rsa_key" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

# 2. Convert the Public Key into JWKS format
data "jwks_from_key" "issuers" {
  key = tls_private_key.rsa_key.public_key_pem

  # Optional Metadata Parameters
  kid = "a-kid"
  use = "sig"
  alg = "RS256"
}

resource "kaleido_platform_application" "oauth_application" {
  name          = "oauth-application"
  admin_enabled = false
  oauth_enabled = true
  oauth = {
    issuer = "my-issuer.example.com"
    aud    = "kaleidoplatform"
    azp    = "my-application"
    # jwks_from_key emits a single JWK object
    jwks              = jsonencode({ keys = [jsondecode(data.jwks_from_key.issuers.jwks)] })
    enable_basic_auth = false // enable if your clients only support basic auth via `x-kld-token:<jwt>`
  }
}

