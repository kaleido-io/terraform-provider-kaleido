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

resource "kaleido_platform_environment" "raas_environment" {
  name = var.environment_name
}

// DA Network
resource "kaleido_platform_network" "da_network" {
  type = "DANetwork"
  name = "da-network"
  environment = kaleido_platform_environment.raas_environment.id
  init_mode = var.da_init_mode
  file_sets = var.da_init_mode == "manual" ? {
    init = {
      files = {
        "genesis.json" = {
          type = "json"
          data = {
          "text" = file(var.da_genesis_file)
        }
        }
      }
    }
  } : {}
  init_files = var.da_init_mode == "manual" ? "init" : ""
  config_json = jsonencode({
    type = "Celestia"
    celestia = {
      type = var.da_network_type
      custom = var.da_network_type == "Custom" ? {
        network = var.da_chain_id
      } : {}
      fundedAccounts = var.funded_account_addresses
    }
  })
}

resource "kaleido_platform_runtime" "celestia_node_runtime" {
  type = "CelestiaNodeRuntime"
  name = "celestia-node-runtime-${count.index}"
  environment = kaleido_platform_environment.raas_environment.id
  size = var.da_node_size
  config_json = jsonencode({})
  count = length(var.da_validator_seeds)
}

resource "kaleido_platform_service" "celestia_node_service" {
  type = "CelestiaNodeService"
  name = "celestia-node-service-${count.index}"
  environment = kaleido_platform_environment.raas_environment.id
  runtime = kaleido_platform_runtime.celestia_node_runtime[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.da_network.id
    }
    seed = length(var.da_validator_seeds) > 1 ? {
      credSetRef = "seed"
    } : {}
    type = var.da_node_type
  })
  cred_sets = length(var.da_validator_seeds) > 1 ? {
    seed = {
      type = "key"
      key = {
        value = var.da_validator_seeds[count.index]
      }
    }
  } : {}
  count = length(var.da_validator_seeds)
}

// L2 Network
resource "kaleido_platform_network" "opl2_network" {
  type = "OPStackL2Network"
  name = var.l2_network_name
  environment = kaleido_platform_environment.raas_environment.id
  config_json = jsonencode({
    type = "Custom"
    custom = {
      gameFactoryAddress = var.l1_game_factory_address
      daConfig = {
        type = var.da_type
        offChain = var.da_type == "OffChain" ? {
          network = {
            id = kaleido_platform_network.da_network.id
          }
          namespace = {
            credSetRef = "namespace"
          }
        } : {}
      }
    },
    l1 = {
      executionURL = var.l1_endpoint
      beaconURL = var.l1_endpoint
    }
  })
  init_mode = "manual"
  init_files = "rollup"
  file_sets = {
    rollup = {
      files = {
        "rollup.json" = {
          type = "json"
          data = {
            "text" = file(var.l2_rollup_file)
          }
        },
        "genesis.json" = {
          type = "json"
          data = {
            "text" = file(var.l2_genesis_file)
          }
        }
      }
    }
  }
  cred_sets = {
    namespace = {
      name = "namespace"
      type = "key"
      key = {
        value = file(var.l2_namespace_file)
      }
    }
  }
}


resource "kaleido_platform_runtime" "opl2_node_runtime" {
  type = "OPSequencerNodeRuntime"
  name = "l2-sequencer-runtime-${count.index+1}"
  environment = kaleido_platform_environment.raas_environment.id
  config_json = jsonencode({})
  size = var.l2_node_size
  count = var.l2_node_count
  depends_on = [kaleido_platform_service.celestia_node_service]
}

resource "kaleido_platform_service" "opl2_node_service" {
  type = "OPSequencerNodeService"
  name = "l2-sequencer-service-${count.index+1}"
  environment = kaleido_platform_environment.raas_environment.id
  runtime = kaleido_platform_runtime.opl2_node_runtime[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.opl2_network.id
    }
    sequencerConfig = {
      sequencerKey = {
        credSetRef = "sequencerKey"
      },
      batcherKey = {
        credSetRef = "batcherKey"
      },
      proposerKey = {
        credSetRef = "proposerKey"
      }
    }
    daConfig = {
      seed = {
        credSetRef = "seed"
      }
    }
  })
  cred_sets = {
    sequencerKey = {
      name = "sequencerKey"
      type = "key"
      key = {
        value = var.l2_sequencer_key
      }
    },
    batcherKey = {
      name = "batcherKey"
      type = "key"
      key = {
        value = var.l2_batcher_key
      }
    },
    proposerKey = {
      name = "proposerKey"
      type = "key"
      key = {
        value = var.l2_proposer_key
      }
    }
    seed = {
      name = "seed"
      type = "key"
      key = {
        value = var.funded_account_seeds[count.index]
      }
    }
  }
  count = var.l2_node_count
  depends_on = [kaleido_platform_service.celestia_node_service]
}


