terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
    }
  }
}

provider "kaleido" {
  platform_api = var.kaleido_platform_api
  platform_username = var.kaleido_platform_username
  platform_password = var.kaleido_platform_password
}

resource "kaleido_platform_environment" "env_0" {
  name = var.environment_name
}

resource "kaleido_platform_group" "group_0" {
  name = var.non_admin_group
}

resource "kaleido_platform_user" "user_0" {
  name = var.non_admin_user_email
  email = var.non_admin_user_email
  sub = var.non_admin_user_sub
  is_admin = false
}

resource "kaleido_platform_group_membership" "group_0" {
  group_id = kaleido_platform_group.group_0.id
  user_id = kaleido_platform_user.user_0.id
}

resource "kaleido_platform_account_access_policy" "aap_0" {
  group_id = kaleido_platform_group.group_0.id
  policy = var.account_access_policy
}


resource "kaleido_platform_runtime" "kmr_0" {
  type = "KeyManager"
  name = "kms1"
  environment = kaleido_platform_environment.env_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "kms_0" {
  type = "KeyManager"
  name = "kms1"
  environment = kaleido_platform_environment.env_0.id
  runtime = kaleido_platform_runtime.kmr_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_wallet" "wallet_0" {
  type = "hdwallet"
  name = "hdwallet1"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.kms_0.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_key" "key_0" {
  name = "key0"
  environment = kaleido_platform_environment.env_0.id
  service = kaleido_platform_service.kms_0.id
  wallet = kaleido_platform_kms_wallet.wallet_0.id
}

resource "kaleido_platform_service_access_policy" "sapol_0" {
  group_id = kaleido_platform_group.group_0.id
  service_id = kaleido_platform_service.kms_0.id
  policy = var.service_access_policy
}

resource "kaleido_platform_service_access" "sapem_0" {
  group_id = kaleido_platform_group.group_0.id
  service_id = kaleido_platform_service.kms_0.id
  permissions_json = jsonencode({
    "api": { 
        "http": [
            {
                "resource": {
                    "matches": "/api/v1/apis/**"
                },
                "actions": ["POST", "GET"]
            },
            {
                "resource": {
                    "matches": "/api/v1/apis"
                },
                "action": "GET"
            }
        ]
    }
})
}

