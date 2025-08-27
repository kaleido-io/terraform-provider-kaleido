terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
    }
  }
}

provider "kaleido" {
  platform_api = var.kaleido_platform_baseurl
  platform_username = var.api_key_name
  platform_password = var.api_key_value
}

resource "kaleido_platform_environment" "canton_env" {
  name = var.environment_name
}

// Create key manager and Wallet
resource "kaleido_platform_runtime" "canton_kms_runtime" {
  type = "KeyManager"
  name = "canton-kms"
  environment = kaleido_platform_environment.canton_env.id
  config_json = jsonencode({})
}

resource "kaleido_platform_service" "canton_kms_service" {
  type = "KeyManager"
  name = "canton-kms"
  environment = kaleido_platform_environment.canton_env.id
  runtime = kaleido_platform_runtime.canton_kms_runtime.id
  config_json = jsonencode({})
}

resource "kaleido_platform_kms_wallet" "canton_wallet" {
  type = "hdwallet"
  name = var.canton_wallet_name
  environment = kaleido_platform_environment.canton_env.id
  service = kaleido_platform_service.canton_kms_service.id
  config_json = jsonencode({})
  depends_on = [kaleido_platform_service.canton_kms_service]
}

// Create a sandbox network
resource "kaleido_platform_network" "sandbox_network" {
  type = "CantonValidatorNetwork"
  name = "canton-sandbox"
  environment = kaleido_platform_environment.canton_env.id
  init_mode = "automated"
  config_json = jsonencode({
    type = "Sandbox"
    sandbox = {
      type = "Local"
    }
  })
}

// super validator node
resource "kaleido_platform_runtime" "canton_sv_runtime" {
  type = "CantonSuperValidatorNodeRuntime"
  name = "canton-sv-node"
  environment = kaleido_platform_environment.canton_env.id
  size = var.sv_node_size
  config_json = jsonencode({})
  depends_on = [kaleido_platform_service.canton_kms_service]
}

resource "kaleido_platform_service" "canton_sv_service" {
  type = "CantonSuperValidatorNodeService"
  name = "canton-sv-node"
  environment = kaleido_platform_environment.canton_env.id
  runtime = kaleido_platform_runtime.canton_sv_runtime.id
  config_json = jsonencode({
    defaultParty = "${kaleido_platform_network.sandbox_network.name}-sv"
    network = {
      id = kaleido_platform_network.sandbox_network.id
    }
    kms = {
      keyManager = {
        id = kaleido_platform_service.canton_kms_service.id
      }
      wallet = kaleido_platform_kms_wallet.canton_wallet.name
    }
  })
  // uncomment `force_delete = true` and run terraform apply before running terraform destory to successfully delete the nodes
  #force_delete = true
}

// Create a synchronizer network
resource "kaleido_platform_network" "synchronizer_network" {
  type = "CantonSynchronizerNetwork"
  name = "canton-synchronizer"
  environment = kaleido_platform_environment.canton_env.id
  init_mode = "automated"
  config_json = jsonencode({
    type = "Local"
  })
  count = var.enable_synchronizer_network ? 1 : 0
}

// synchronizer node
resource "kaleido_platform_runtime" "canton_synchronizer_runtime" {
  type = "CantonSynchronizerNodeRuntime"
  name = "canton-synchronizer-node"
  environment = kaleido_platform_environment.canton_env.id
  size = var.synchronizer_node_size
  config_json = jsonencode({})
  depends_on = [kaleido_platform_service.canton_kms_service]
  count = var.enable_synchronizer_network ? 1 : 0
}

resource "kaleido_platform_service" "canton_synchronizer_service" {
  type = "CantonSynchronizerNodeService"
  name = "canton-synchronizer-node"
  environment = kaleido_platform_environment.canton_env.id
  runtime = kaleido_platform_runtime.canton_synchronizer_runtime[0].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.synchronizer_network[0].id
    }
    kms = {
      keyManager = {
        id = kaleido_platform_service.canton_kms_service.id
      }
      wallet = kaleido_platform_kms_wallet.canton_wallet.name
    }
  })
  count = var.enable_synchronizer_network ? 1 : 0
}

// participant node
resource "kaleido_platform_runtime" "canton_participant_runtime" {
  type = "CantonParticipantNodeRuntime"
  name = "canton-participant-node"
  environment = kaleido_platform_environment.canton_env.id
  size = var.participant_node_size
  config_json = jsonencode({})
  depends_on = [kaleido_platform_service.canton_kms_service, kaleido_platform_service.canton_sv_service]
}

resource "kaleido_platform_service" "canton_participant_service" {
  type = "CantonParticipantNodeService"
  name = "canton-participant-node"
  environment = kaleido_platform_environment.canton_env.id
  runtime = kaleido_platform_runtime.canton_participant_runtime.id
  config_json = jsonencode({
    defaultParty = "${kaleido_platform_network.sandbox_network.name}-participant"
    validatorNetwork = {
        id = kaleido_platform_network.sandbox_network.id
    }
    synchronizerNetworks = var.enable_synchronizer_network ? [
      {
        network = {
          id = kaleido_platform_network.synchronizer_network[0].id
        }
      }
    ] : []
    kms = {
      keyManager = {
        id = kaleido_platform_service.canton_kms_service.id
      }
      wallet = kaleido_platform_kms_wallet.canton_wallet.name
    }
  })
  depends_on = [kaleido_platform_service.canton_kms_service, kaleido_platform_service.canton_sv_service]
}

// hostnames for participant ledger and admin
resource "kaleido_platform_hostname" "canton_participant_ledger_hostname" {
  environment = kaleido_platform_environment.canton_env.id
  service = kaleido_platform_service.canton_participant_service.id
  name = "canton-participant-hostname"
  hostname = "participant-ledger"
  endpoints = ["ledger"]
  mtls = false
}

resource "kaleido_platform_hostname" "canton_participant_admin_hostname" {
  environment = kaleido_platform_environment.canton_env.id
  service = kaleido_platform_service.canton_participant_service.id
  name = "canton-participant-hostname"
  hostname = "participant-admin"
  endpoints = ["admin"]
  mtls = false
}
