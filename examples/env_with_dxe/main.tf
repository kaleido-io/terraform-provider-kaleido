/*
This creates suite of environments using all available
environment types and consensus methods.
*/
terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
    }
  }
}

provider "kaleido" {
  api = "https://console${var.kaleido_region}.kaleido.io/api/v1"
  api_key = "${var.kaleido_api_key}"
}

/*
This represents a Consortia. Give it a name and a description.
"mode" can be set to "single-org" or ...
*/
resource "kaleido_consortium" "consortium" {
  name = "${var.consortium_name}"
  description = "${var.network_description}"
}

/*
This creates a membership for each node
*/
resource "kaleido_membership" "member" {
  count = "${var.node_count}"
  consortium_id = "${kaleido_consortium.consortium.id}"
  org_name = "Org ${count.index + 1}"
}

/*
Create an environment
*/
resource "kaleido_environment" "env" {
  consortium_id = "${kaleido_consortium.consortium.id}"
  multi_region = "${var.multi_region}"
  name = "${var.env_name}"
  env_type = "${var.provider_type}"
  consensus_type = "${var.consensus}"
  description = "${var.env_description}"
}

/*
Create nodes
*/
resource "kaleido_node" "kaleido" {
  count = "${var.node_count}"
  consortium_id = "${kaleido_consortium.consortium.id}"
  environment_id = "${kaleido_environment.env.id}"
  membership_id = "${element(kaleido_membership.member.*.id, count.index)}"
  name = "Node ${count.index + 1}"
  size = "${var.node_size}"
}

/*
Create ipfs service
*/
resource "kaleido_service" "kaleido" {
  count = "${var.service_count}"
  consortium_id = "${kaleido_consortium.consortium.id}"
  environment_id = "${kaleido_environment.env.id}"
  membership_id = "${element(kaleido_membership.member.*.id, count.index)}"
  name = "IPFS ${count.index + 1}"
  service_type = "ipfs"

  depends_on = ["kaleido_node.kaleido"]
}

/*
Create DXE service - please contact Kaleido support if you're interested in using DataExchange Enterprise for FireFly multi-party networks
https://hyperledger.github.io/firefly/v1.2.2/overview/multiparty/data_exchange.html
*/
resource "kaleido_service" "dxe" {
  count = "${var.service_count}"
  consortium_id = "${kaleido_consortium.consortium.id}"
  environment_id = "${kaleido_environment.env.id}"
  membership_id = "${element(kaleido_membership.member.*.id, count.index)}"
  service_type = "dataexchange"
  name = "DXE ${count.index + 1}"
  size = "small"
}