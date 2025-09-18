// Generates a SECP256K1 key pair for a Besu node to use for P2P and block proposals
resource "kaleido_platform_besu_node_key" "besu_node_key" {}

// Provide the private key securely to the Besu node in the config_json via a cred set
resource "kaleido_platform_service" "besu_node" {
  type        = "BesuNode"
  name        = "a-besu-node"
  environment = "e:1234abcd"
  runtime     = "r:1234abcd"
  config_json = jsonencode({
    network = {
      id = "n:1234abcd"
    }
    nodeKey = {
      credSetRef = "nodeKey"
    }
    signer = true // Node will ask the network to vote it in as a validator
  })
  cred_sets = {
    nodeKey = {
      type = "key"
      key = {
        value = kaleido_platform_besu_node_key.besu_node_key.private_key
      }
    }
  }
}
