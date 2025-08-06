resource "kaleido_network_connector" "requestor" {
  name = "${var.requestor_name}_to_${var.acceptor_name}"
  type = "Platform"
  network = var.requestor_network_id
  environment = var.requestor_environment_id
  zone = var.node_peerable_zone
  platform_requestor = {
    target_account_id = var.account_id
    target_environment_id = var.acceptor_environment_id
    target_network_id = var.acceptor_network_id
  }
}

resource "kaleido_network_connector" "acceptor" {
  name = "${var.acceptor_name}_from_${var.requestor_name}"
  type = "Platform"
  network = var.acceptor_network_id
  environment = var.acceptor_environment_id
  zone = var.node_peerable_zone
  platform_acceptor = {
    target_account_id = var.account_id
    target_environment_id = var.requestor_environment_id
    target_network_id = var.requestor_network_id
    target_connector_id = kaleido_network_connector.requestor.id
  }
}