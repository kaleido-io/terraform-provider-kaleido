/*
This creates suite of environments using all available
environment types and consensus methods.
*/

provider "kaleido" {
  "api" = "${var.kaleido_api_url}"
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
  org_name = "Org ${count.index + 1}"
}

/*
This whitelists a region into the consortium for deployment
This is only required if deploying to a zone that is not the home zone
of the API URL - otherwise you will get a 409 conflict on deployment
*/
resource "kaleido_czone" "allowed_region" {
  consortium_id = "${kaleido_consortium.consortium.id}"
  cloud = "${var.cloud}"
  region = "${var.region}"
}

/*
Create an environment
*/
resource "kaleido_environment" "env" {
  consortium_id = "${kaleido_consortium.consortium.id}"
  multi_region = true
  name = "${var.env_name}"
  env_type = "${var.provider}"
  consensus_type = "${var.consensus}"
  description = "${var.env_description}"
}

/*
This creates the first deployment zone for your environment.
For this sample we refer to it explicitly when deploying nodes, showing how
you could have multiple deployment zones in a single environment if required
*/
resource "kaleido_ezone" "deployment_zone" {
  consortium_id = "${kaleido_consortium.consortium.id}"
  environment_id = "${kaleido_environment.env.id}"
  cloud = "${var.cloud}"
  region = "${var.region}"
}

/*
Create nodes
*/
resource "kaleido_node" "kaleido" {
  count = "${var.node_count}"
  consortium_id = "${kaleido_consortium.consortium.id}"
  environment_id = "${kaleido_environment.env.id}"
  membership_id = "${element(kaleido_membership.member.*.id, count.index)}"
  zone_id = "${kaleido_ezone.deployment_zone.id}"
  name = "Node ${count.index + 1}"
  size = "${var.node_size}"
}
