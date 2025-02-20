terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
      version = "1.1.0-rc.4"
      configuration_aliases = [ kaleido.originator, kaleido.secondary ]
    }
  }
}

provider "kaleido" {
  alias = "originator"
  platform_api = var.originator_api_url
  platform_username = var.originator_api_key_name
  platform_password = var.originator_api_key_value
}

provider "kaleido" {
  alias = "secondary"
  platform_api = var.secondary_api_url
  platform_username = var.secondary_api_key_name
  platform_password = var.secondary_api_key_value
}

// Environment 1 - the originator of the network

resource "kaleido_platform_environment" "env_og" {
  provider = kaleido.originator
  name = var.originator_name
}

resource "kaleido_platform_network" "net_og" {
  provider = kaleido.originator
  type = "BesuNetwork"
  name = var.originator_name
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


resource "kaleido_platform_runtime" "bnr_signer_net_og" {
  provider = kaleido.originator
  type = "BesuNode"
  name = "${var.originator_name}_signer${count.index+1}"
  environment = kaleido_platform_environment.env_og.id
  config_json = jsonencode({})
  count = var.originator_signer_count
}

resource "kaleido_platform_service" "bns_signer_net_og" {
  provider = kaleido.originator
  type = "BesuNode"
  name = "${var.originator_name}_signer${count.index+1}"
  environment = kaleido_platform_environment.env_og.id
  runtime = kaleido_platform_runtime.bnr_signer_net_og[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.net_og.id
    }
    signer = true
  })
  count = var.originator_signer_count
}

resource "kaleido_platform_runtime" "bnr_peer_net_og" {
  provider = kaleido.originator
  type = "BesuNode"
  name = "${var.originator_name}_peer${count.index+1}"
  environment = kaleido_platform_environment.env_og.id
  zone = var.originator_peer_network_dz
  config_json = jsonencode({})
  count = var.originator_peer_count
}

resource "kaleido_platform_service" "bns_peer_net_og" {
  provider = kaleido.originator
  type = "BesuNode"
  name = "${var.originator_name}_peer${count.index+1}"
  environment = kaleido_platform_environment.env_og.id
  runtime = kaleido_platform_runtime.bnr_peer_net_og[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.net_og.id
    }
    signer = false
  })
  count = var.originator_peer_count
}

resource "kaleido_platform_runtime" "gwr_net_og" {
  provider = kaleido.originator
  type = "EVMGateway"
  name = "${var.originator_name}_gateway"
  environment = kaleido_platform_environment.env_og.id
  config_json = jsonencode({})
  count = var.originator_gateway_count
}


resource "kaleido_platform_service" "gws_net_og" {
  provider = kaleido.originator
  type = "EVMGateway"
  name = "${var.originator_name}_gateway"
  environment = kaleido_platform_environment.env_og.id
  runtime = kaleido_platform_runtime.gwr_net_og[count.index].id
  config_json = jsonencode({
    network = {
      id =  kaleido_platform_network.net_og.id
    }
  })
  count = var.originator_gateway_count
}

data "kaleido_platform_network_bootstrap_data" "net_og_bootstrap" {
  provider = kaleido.originator
  environment = kaleido_platform_environment.env_og.id
  network = kaleido_platform_network.net_og.id
  depends_on = [
    kaleido_platform_service.bns_signer_net_og,
    kaleido_platform_network.net_og
  ]
}



// Environment 2 - another member of the network

resource "kaleido_platform_environment" "env_sec" {
  provider = kaleido.secondary
  name = var.secondary_name
}

resource "kaleido_platform_network" "net_sec" {
  provider = kaleido.secondary
  type = "BesuNetwork"
  name = var.secondary_name
  environment = kaleido_platform_environment.env_sec.id
  init_mode = "manual"
  file_sets = data.kaleido_platform_network_bootstrap_data.net_og_bootstrap.bootstrap_files != null ? {
    init = data.kaleido_platform_network_bootstrap_data.net_og_bootstrap.bootstrap_files
  } : {}
  init_files = data.kaleido_platform_network_bootstrap_data.net_og_bootstrap.bootstrap_files != null ? "init" : null
  config_json = jsonencode({})
  depends_on = [kaleido_platform_network.net_og, kaleido_platform_service.bns_signer_net_og, data.kaleido_platform_network_bootstrap_data.net_og_bootstrap]
}

resource "kaleido_platform_runtime" "bnr_peer_net_sec" {
  provider = kaleido.secondary
  type = "BesuNode"
  name = "${var.secondary_name}_peer${count.index+1}"
  environment = kaleido_platform_environment.env_sec.id
  zone = var.secondary_peer_network_dz
  config_json = jsonencode({})
  count = var.secondary_peer_count
  depends_on = [kaleido_platform_network.net_sec] 
}

resource "kaleido_platform_service" "bns_peer_net_sec" {
  provider = kaleido.secondary
  type = "BesuNode"
  name = "${var.secondary_name}_peer${count.index+1}"
  environment = kaleido_platform_environment.env_sec.id
  runtime = kaleido_platform_runtime.bnr_peer_net_sec[count.index].id
  config_json = jsonencode({
    network = {
      id = kaleido_platform_network.net_sec.id
    }
    signer = false
  })
  count = var.secondary_peer_count
}

resource "kaleido_platform_runtime" "gwr_net_sec" {
  provider = kaleido.secondary
  type = "EVMGateway"
  name = "${var.originator_name}_gateway"
  environment = kaleido_platform_environment.env_sec.id
  config_json = jsonencode({})
  depends_on = [kaleido_platform_network.net_sec]
  count = var.secondary_gateway_count
}

resource "kaleido_platform_service" "gws_net_sec" {
  provider = kaleido.secondary
  type = "EVMGateway"
  name = "${var.originator_name}_gateway"
  environment = kaleido_platform_environment.env_sec.id
  runtime = kaleido_platform_runtime.gwr_net_sec[count.index].id
  config_json = jsonencode({
    network = {
      id =  kaleido_platform_network.net_sec.id
    }
  })
  count = var.secondary_gateway_count
}


// Network Connectors
resource "kaleido_network_connector" "net_sec_connector" {
  provider = kaleido.secondary
  type = "Permitted"
  name = "${var.secondary_name}_conn"
  environment = kaleido_platform_environment.env_sec.id
  network = kaleido_platform_network.net_sec.id
  zone = var.secondary_peer_network_dz
  permitted_json = jsonencode({ peers = [ for peer in resource.kaleido_platform_service.bns_peer_net_og : jsondecode(peer.connectivity_json) ] })
  depends_on = [kaleido_platform_network.net_sec, kaleido_platform_service.bns_peer_net_og]
}


resource "kaleido_network_connector" "net_og_connector" {
  provider = kaleido.originator
  type = "Permitted"
  name = "${var.originator_name}_conn"
  environment = kaleido_platform_environment.env_og.id
  network = kaleido_platform_network.net_og.id
  zone = var.originator_peer_network_dz
  permitted_json = jsonencode({ peers = [ for peer in resource.kaleido_platform_service.bns_peer_net_sec : jsondecode(peer.connectivity_json) ] })
  depends_on = [kaleido_platform_network.net_og, kaleido_platform_service.bns_peer_net_sec]
}
