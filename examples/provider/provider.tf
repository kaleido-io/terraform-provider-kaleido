terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
      version = ">1.2.0"
    }
  }
}

provider "kaleido" {
  alias = "provider"
  platform_api = var.kaleido_platform_api                 # https://<account-name>.${PLATFORM_DOMAIN}
  platform_username = var.kaleido_platform_username       # the name of the API key
  platform_password = var.kaleido_platform_password       # the secret of the API key
}

provider "kaleido" {
  alias = "baas_provider"
  api = "https://console.kaleido.io/api/v1"
  api_key = var.kaleido_api_key                           # the API key for the Kaleido BaaS user
}