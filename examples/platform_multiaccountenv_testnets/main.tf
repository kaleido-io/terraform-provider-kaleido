

terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
      configuration_aliases = [kaleido.originator, kaleido.joiner_one, kaleido.joiner_two]
    }
    random = {
      source  = "hashicorp/random"
      version = "3.6.3"
    }
  }
}

provider "kaleido" {
  alias = "originator"
  platform_api = var.originator_api_url
  platform_username = var.originator_api_key_name
  platform_password = var.originator_api_key_value
}

data "kaleido_platform_account" "acct_og" {
    provider = kaleido.originator
}

provider "kaleido" {
  alias = "joiner_one"
  platform_api = var.joiner_one_api_url
  platform_username = var.joiner_one_api_key_name
  platform_password = var.joiner_one_api_key_value
}

data "kaleido_platform_account" "acct_j1" {
  provider = kaleido.joiner_one
}

provider "kaleido" {
  alias = "joiner_two"
  platform_api = var.joiner_two_api_url
  platform_username = var.joiner_two_api_key_name
  platform_password = var.joiner_two_api_key_value
}

data "kaleido_platform_account" "acct_j2" {
  provider = kaleido.joiner_two
}

provider "random" {}

// Environment 1 - the originator of the testnets

resource "kaleido_platform_environment" "env_og" {
  provider = kaleido.originator
  name = var.originator_name
}


resource "kaleido_platform_stack" "chain_infra_besu_stack_og" {
  provider = kaleido.originator
  environment = kaleido_platform_environment.env_og.id
  name = "chain_infra_besu_stack"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.besunet_og.id
}

resource "kaleido_platform_stack" "chain_infra_ipfs_stack_og" {
  provider = kaleido.originator
  environment = kaleido_platform_environment.env_og.id
  name = "chain_infra_ipfs_stack"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.ipfsnet_og.id
}

resource "kaleido_platform_network" "besunet_og" {
  provider = kaleido.originator
  type = "BesuNetwork"
  name = var.besu_testnet_name
  environment = kaleido_platform_environment.env_og.id
  init_mode = "automated"

  config_json = jsonencode({
    chainID = 12345
    bootstrapOptions = {
      qbft = {
        blockperiodseconds = 5
      }
      eipBlockConfig = {
        shanghaiTime = 0
      }
      initialBalances = {
        "0x12F62772C4652280d06E64CfBC9033d409559aD4" = "0x111111111111",
      }
      blockConfigFlags = {
        zeroBaseFee = true
      }
    }
  })
}

resource "random_id" "swarm_key_og" {
  byte_length = 32
}

resource "kaleido_platform_network" "ipfsnet_og" {
  provider = kaleido.originator
  type = "IPFSNetwork"
  name = var.ipfs_testnet_name
  environment = kaleido_platform_environment.env_og.id
  config_json = jsonencode({
    swarmKey = random_id.swarm_key_og.hex
  })
}

resource "kaleido_platform_runtime" "bnr_signer_net_og" {
  provider = kaleido.originator
  type = "BesuNode"
  name = "${var.originator_name}_signer${count.index+1}"
  environment = kaleido_platform_environment.env_og.id
  config_json = jsonencode({})
  count = var.originator_signer_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_og.id
}

resource "kaleido_platform_service" "bns_signer_net_og" {
  provider = kaleido.originator
  type = "BesuNode"
  name = "${var.originator_name}_signer${count.index+1}"
  environment = kaleido_platform_environment.env_og.id
  runtime = kaleido_platform_runtime.bnr_signer_net_og[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.besunet_og.id
    }
    signer = true
  })
  count = var.originator_signer_count
  force_delete = true
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_og.id
}

resource "kaleido_platform_runtime" "bnr_peer_net_og" {
  provider = kaleido.originator
  type = "BesuNode"
  name = "${var.originator_name}_peer${count.index+1}"
  environment = kaleido_platform_environment.env_og.id
  zone = var.originator_peer_network_dz
  config_json = jsonencode({})
  count = var.originator_peer_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_og.id
}

resource "kaleido_platform_service" "bns_peer_net_og" {
  provider = kaleido.originator
  type = "BesuNode"
  name = "${var.originator_name}_peer${count.index+1}"
  environment = kaleido_platform_environment.env_og.id
  runtime = kaleido_platform_runtime.bnr_peer_net_og[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.besunet_og.id
    }
    signer = false
  })
  count = var.originator_peer_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_og.id
}

resource "kaleido_platform_runtime" "gwr_net_og" {
  provider = kaleido.originator
  type = "EVMGateway"
  name = "${var.originator_name}_gateway"
  environment = kaleido_platform_environment.env_og.id
  config_json = jsonencode({})
  count = var.originator_gateway_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_og.id
}


resource "kaleido_platform_service" "gws_net_og" {
  provider = kaleido.originator
  type = "EVMGateway"
  name = "${var.originator_name}_gateway"
  environment = kaleido_platform_environment.env_og.id
  runtime = kaleido_platform_runtime.gwr_net_og[count.index].id
  config_json = jsonencode({
    network = {
      id =  kaleido_platform_network.besunet_og.id
    }
  })
  count = var.originator_gateway_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_og.id
}

data "kaleido_platform_network_bootstrap_data" "net_og_bootstrap" {
  provider = kaleido.originator
  environment = kaleido_platform_environment.env_og.id
  network = kaleido_platform_network.besunet_og.id
  depends_on = [
    kaleido_platform_service.bns_signer_net_og,
    kaleido_platform_network.besunet_og
  ]
}


resource "kaleido_platform_runtime" "inr_net_og" {
  provider = kaleido.originator

  type = "IPFSNode"
  name = var.originator_name
  zone = var.originator_peer_network_dz
  environment = kaleido_platform_environment.env_og.id
  config_json = jsonencode({ })
  stack_id = kaleido_platform_stack.chain_infra_ipfs_stack_og.id
}

resource "kaleido_platform_service" "ins_net_og" {
  provider = kaleido.originator

  type = "IPFSNode"
  name = var.originator_name
  environment = kaleido_platform_environment.env_og.id
  runtime = kaleido_platform_runtime.inr_net_og.id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.ipfsnet_og.id
    }
  })
  stack_id = kaleido_platform_stack.chain_infra_ipfs_stack_og.id
}


// Environment 2 - another member of the testnets joining
resource "kaleido_platform_stack" "chain_infra_besu_stack_j1" {
  provider = kaleido.joiner_one
  environment = kaleido_platform_environment.env_j1.id
  name = "chain_infra_besu_stack"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.besunet_j1.id
}

resource "kaleido_platform_stack" "chain_infra_ipfs_stack_j1" {
  provider = kaleido.joiner_one
  environment = kaleido_platform_environment.env_j1.id
  name = "chain_infra_ipfs_stack"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.ipfsnet_j1.id
}

resource "kaleido_platform_environment" "env_j1" {
  provider = kaleido.joiner_one
  name = var.joiner_one_name
}

resource "kaleido_platform_network" "besunet_j1" {
  provider = kaleido.joiner_one
  type = "BesuNetwork"
  name = var.joiner_one_name
  environment = kaleido_platform_environment.env_j1.id
  init_mode = "manual"
  file_sets = data.kaleido_platform_network_bootstrap_data.net_og_bootstrap.bootstrap_files != null ? {
    init = data.kaleido_platform_network_bootstrap_data.net_og_bootstrap.bootstrap_files
  } : {}
  init_files = data.kaleido_platform_network_bootstrap_data.net_og_bootstrap.bootstrap_files != null ? "init" : null
  config_json = jsonencode({})
  depends_on = [kaleido_platform_network.besunet_og, kaleido_platform_service.bns_signer_net_og, data.kaleido_platform_network_bootstrap_data.net_og_bootstrap]
}

resource "kaleido_platform_network" "ipfsnet_j1" {
  provider = kaleido.joiner_one
  type = "IPFSNetwork"
  name = var.ipfs_testnet_name
  environment = kaleido_platform_environment.env_j1.id
  config_json = jsonencode({
    swarmKey = random_id.swarm_key_og.hex
  })
}

resource "kaleido_platform_runtime" "bnr_peer_net_j1" {
  provider = kaleido.joiner_one
  type = "BesuNode"
  name = "${var.joiner_one_name}_peer${count.index+1}"
  environment = kaleido_platform_environment.env_j1.id
  zone = var.joiner_one_peer_network_dz
  config_json = jsonencode({})
  count = var.joiner_one_peer_count
  depends_on = [kaleido_platform_network.besunet_j1]
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_j1.id
}

resource "kaleido_platform_service" "bns_peer_net_j1" {
  provider = kaleido.joiner_one
  type = "BesuNode"
  name = "${var.joiner_one_name}_peer${count.index+1}"
  environment = kaleido_platform_environment.env_j1.id
  runtime = kaleido_platform_runtime.bnr_peer_net_j1[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.besunet_j1.id
    }
    signer = false
  })
  count = var.joiner_one_peer_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_j1.id
}

resource "kaleido_platform_runtime" "gwr_net_j1" {
  provider = kaleido.joiner_one
  type = "EVMGateway"
  name = "${var.joiner_one_name}_gateway"
  environment = kaleido_platform_environment.env_j1.id
  config_json = jsonencode({})
  depends_on = [kaleido_platform_network.besunet_j1]
  count = var.joiner_one_gateway_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_j1.id
}

resource "kaleido_platform_service" "gws_net_j1" {
  provider = kaleido.joiner_one
  type = "EVMGateway"
  name = "${var.joiner_one_name}_gateway"
  environment = kaleido_platform_environment.env_j1.id
  runtime = kaleido_platform_runtime.gwr_net_j1[count.index].id
  config_json = jsonencode({
    network = {
      id =  kaleido_platform_network.besunet_j1.id
    }
  })
  count = var.joiner_one_gateway_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_j1.id
}

resource "kaleido_platform_runtime" "inr_net_j1" {
  provider = kaleido.joiner_one

  type = "IPFSNode"
  name = var.joiner_one_name
  zone = var.joiner_one_peer_network_dz
  environment = kaleido_platform_environment.env_j1.id
  config_json = jsonencode({ })
  stack_id = kaleido_platform_stack.chain_infra_ipfs_stack_j1.id
}

resource "kaleido_platform_service" "ins_net_j1" {
  provider = kaleido.joiner_one

  type = "IPFSNode"
  name = var.joiner_one_name
  environment = kaleido_platform_environment.env_j1.id
  runtime = kaleido_platform_runtime.inr_net_j1.id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.ipfsnet_j1.id
    }
  })
  stack_id = kaleido_platform_stack.chain_infra_ipfs_stack_j1.id
}

// Environment 3 - another member of the testnets joining
resource "kaleido_platform_stack" "chain_infra_besu_stack_j2" {
  provider = kaleido.joiner_two
  environment = kaleido_platform_environment.env_j2.id
  name = "chain_infra_besu_stack"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.besunet_j2.id
}

resource "kaleido_platform_stack" "chain_infra_ipfs_stack_j2" {
  provider = kaleido.joiner_two
  environment = kaleido_platform_environment.env_j2.id
  name = "chain_infra_ipfs_stack"
  type = "chain_infrastructure"
  network_id = kaleido_platform_network.ipfsnet_j2.id
}


resource "kaleido_platform_environment" "env_j2" {
  provider = kaleido.joiner_two
  name = var.joiner_two_name
}

resource "kaleido_platform_network" "besunet_j2" {
  provider = kaleido.joiner_two
  type = "BesuNetwork"
  name = var.joiner_two_name
  environment = kaleido_platform_environment.env_j2.id
  init_mode = "manual"
  file_sets = data.kaleido_platform_network_bootstrap_data.net_og_bootstrap.bootstrap_files != null ? {
    init = data.kaleido_platform_network_bootstrap_data.net_og_bootstrap.bootstrap_files
  } : {}
  init_files = data.kaleido_platform_network_bootstrap_data.net_og_bootstrap.bootstrap_files != null ? "init" : null
  config_json = jsonencode({})
  depends_on = [kaleido_platform_network.besunet_og, kaleido_platform_service.bns_signer_net_og, data.kaleido_platform_network_bootstrap_data.net_og_bootstrap]
}

resource "kaleido_platform_network" "ipfsnet_j2" {
  provider = kaleido.joiner_two
  type = "IPFSNetwork"
  name = var.ipfs_testnet_name
  environment = kaleido_platform_environment.env_j2.id
  config_json = jsonencode({
    swarmKey = random_id.swarm_key_og.hex
  })
}

resource "kaleido_platform_runtime" "bnr_peer_net_j2" {
  provider = kaleido.joiner_two
  type = "BesuNode"
  name = "${var.joiner_two_name}_peer${count.index+1}"
  environment = kaleido_platform_environment.env_j2.id
  zone = var.joiner_two_peer_network_dz
  config_json = jsonencode({})
  count = var.joiner_two_peer_count
  depends_on = [kaleido_platform_network.besunet_j2]
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_j2.id
}

resource "kaleido_platform_service" "bns_peer_net_j2" {
  provider = kaleido.joiner_two
  type = "BesuNode"
  name = "${var.joiner_two_name}_peer${count.index+1}"
  environment = kaleido_platform_environment.env_j2.id
  runtime = kaleido_platform_runtime.bnr_peer_net_j2[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.besunet_j2.id
    }
    signer = false
  })
  count = var.joiner_two_peer_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_j2.id
}

resource "kaleido_platform_runtime" "gwr_net_j2" {
  provider = kaleido.joiner_two
  type = "EVMGateway"
  name = "${var.joiner_two_name}_gateway"
  environment = kaleido_platform_environment.env_j2.id
  config_json = jsonencode({})
  depends_on = [kaleido_platform_network.besunet_j2]
  count = var.joiner_two_gateway_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_j2.id
}

resource "kaleido_platform_service" "gws_net_j2" {
  provider = kaleido.joiner_two
  type = "EVMGateway"
  name = "${var.joiner_two_name}_gateway"
  environment = kaleido_platform_environment.env_j2.id
  runtime = kaleido_platform_runtime.gwr_net_j2[count.index].id
  config_json = jsonencode({
    network = {
      id =  kaleido_platform_network.besunet_j2.id
    }
  })
  count = var.joiner_two_gateway_count
  stack_id = kaleido_platform_stack.chain_infra_besu_stack_j2.id
}

resource "kaleido_platform_runtime" "inr_net_j2" {
  provider = kaleido.joiner_two

  type = "IPFSNode"
  name = var.joiner_two_name
  zone = var.joiner_two_peer_network_dz
  environment = kaleido_platform_environment.env_j2.id
  config_json = jsonencode({ })
  stack_id = kaleido_platform_stack.chain_infra_ipfs_stack_j2.id
}

resource "kaleido_platform_service" "ins_net_j2" {
  provider = kaleido.joiner_two

  type = "IPFSNode"
  name = var.joiner_two_name
  environment = kaleido_platform_environment.env_j2.id
  runtime = kaleido_platform_runtime.inr_net_j2.id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.ipfsnet_j2.id
    }
  })
  stack_id = kaleido_platform_stack.chain_infra_ipfs_stack_j2.id
}

// Besu Network Connectors
resource "kaleido_network_connector" "besunet_j1_connector_og" {
  provider = kaleido.joiner_one
  type = "Platform"
  name = "${var.joiner_one_name}_conn_${var.originator_name}"
  environment = kaleido_platform_environment.env_j1.id
  network = kaleido_platform_network.besunet_j1.id
  zone = var.joiner_one_peer_network_dz

  platform_requestor = {
    target_account_id = data.kaleido_platform_account.acct_og.account_id
    target_environment_id = kaleido_platform_environment.env_og.id
    target_network_id = kaleido_platform_network.besunet_og.id
  }
}


resource "kaleido_network_connector" "besunet_og_connector_j1" {
  provider = kaleido.originator
  type = "Platform"
  name = "${var.originator_name}_conn_${var.joiner_one_name}"
  environment = kaleido_platform_environment.env_og.id
  network = kaleido_platform_network.besunet_og.id
  zone = var.originator_peer_network_dz

  platform_acceptor = {
    target_account_id = data.kaleido_platform_account.acct_j1.account_id
    target_environment_id = kaleido_platform_environment.env_j1.id
    target_network_id = kaleido_platform_network.besunet_j1.id
    target_connector_id = kaleido_network_connector.besunet_j1_connector_og.id
  }
}

resource "kaleido_network_connector" "besunet_og_connector_j2" {
  provider = kaleido.originator
  type = "Platform"
  name = "${var.originator_name}_conn_${var.joiner_two_name}"
  environment = kaleido_platform_environment.env_og.id
  network = kaleido_platform_network.besunet_og.id
  zone = var.originator_peer_network_dz

  platform_requestor = {
    target_account_id = data.kaleido_platform_account.acct_j2.account_id
    target_environment_id = kaleido_platform_environment.env_j2.id
    target_network_id = kaleido_platform_network.besunet_j2.id
  }
}

resource "kaleido_network_connector" "besunet_j2_connector_og" {
  provider = kaleido.joiner_two
  type = "Platform"
  name = "${var.joiner_two_name}_conn_${var.originator_name}"
  environment = kaleido_platform_environment.env_j2.id
  network = kaleido_platform_network.besunet_j2.id
  zone = var.joiner_two_peer_network_dz

  platform_acceptor = {
    target_account_id = data.kaleido_platform_account.acct_og.account_id
    target_environment_id = kaleido_platform_environment.env_og.id
    target_network_id = kaleido_platform_network.besunet_og.id
    target_connector_id = kaleido_network_connector.besunet_og_connector_j2.id
  }
}

// IPFS Network Connectors
resource "kaleido_network_connector" "ipfsnet_j1_connector_og" {
  provider = kaleido.joiner_one
  type = "Platform"
  name = "${var.joiner_one_name}_conn_${var.originator_name}"
  environment = kaleido_platform_environment.env_j1.id
  network = kaleido_platform_network.ipfsnet_j1.id
  zone = var.joiner_one_peer_network_dz

  platform_requestor = {
    target_account_id = data.kaleido_platform_account.acct_og.account_id
    target_environment_id = kaleido_platform_environment.env_og.id
    target_network_id = kaleido_platform_network.ipfsnet_og.id
  }
}


resource "kaleido_network_connector" "ipfsnet_og_connector_j1" {
  provider = kaleido.originator
  type = "Platform"
  name = "${var.originator_name}_conn_${var.joiner_one_name}"
  environment = kaleido_platform_environment.env_og.id
  network = kaleido_platform_network.ipfsnet_og.id
  zone = var.originator_peer_network_dz

  platform_acceptor = {
    target_account_id = data.kaleido_platform_account.acct_j1.account_id
    target_environment_id = kaleido_platform_environment.env_j1.id
    target_network_id = kaleido_platform_network.ipfsnet_j1.id
    target_connector_id = kaleido_network_connector.ipfsnet_j1_connector_og.id
  }
}

resource "kaleido_network_connector" "ipfsnet_og_connector_j2" {
  provider = kaleido.originator
  type = "Platform"
  name = "${var.originator_name}_conn_${var.joiner_two_name}"
  environment = kaleido_platform_environment.env_og.id
  network = kaleido_platform_network.ipfsnet_og.id
  zone = var.originator_peer_network_dz

  platform_requestor = {
    target_account_id = data.kaleido_platform_account.acct_j2.account_id
    target_environment_id = kaleido_platform_environment.env_j2.id
    target_network_id = kaleido_platform_network.ipfsnet_j2.id
  }
}

resource "kaleido_network_connector" "ipfsnet_j2_connector_og" {
  provider = kaleido.joiner_two
  type = "Platform"
  name = "${var.joiner_two_name}_conn_${var.originator_name}"
  environment = kaleido_platform_environment.env_j2.id
  network = kaleido_platform_network.ipfsnet_j2.id
  zone = var.joiner_two_peer_network_dz

  platform_acceptor = {
    target_account_id = data.kaleido_platform_account.acct_og.account_id
    target_environment_id = kaleido_platform_environment.env_og.id
    target_network_id = kaleido_platform_network.ipfsnet_og.id
    target_connector_id = kaleido_network_connector.ipfsnet_og_connector_j2.id
  }
}
