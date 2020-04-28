/*
This creates suite of environments using all available
environment types and consensus methods.
*/

provider "kaleido" {
  "api" = "https://console${var.kaleido_region}.kaleido.io/api/v1"
  "api_key" = "${var.kaleido_api_key}"
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
  org_name = "Node ${count.index + 1}"
}

/*
Create an environment
*/
resource "kaleido_environment" "env" {
  consortium_id = "${kaleido_consortium.consortium.id}"
  multi_region = "${var.multi_region}"
  name = "${var.env_name}"
  env_type = "${var.provider}"
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
