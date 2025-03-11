terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
      version = "1.1.0"
    }
  }
}

provider "kaleido" {
  platform_api = var.kaleido_platform_api
  platform_username = var.kaleido_platform_username
  platform_password = var.kaleido_platform_password
}
