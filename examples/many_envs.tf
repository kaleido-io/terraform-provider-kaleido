/*
This creates suite of environments using all available
environment types and consensus methods.
*/

provider "kaleido" {
  "api" = "https://console.kaleido.cloud/api/v1"
  "api_key" = "XXXXX"
}

variable "env_types" {
  type = "list"
  default = ["quorum"]
  description = "List of environment types you want to deploy. Options are 'quorum' and 'geth'."
}

variable "quorum_consensus" {
  type = "list"
  default = ["raft", "ibft"]
  description = "Consensus methods supported by quorum."
}

variable "consortia_count" {
  type = "string"
  default = 1
  description = "The number of consortia to deploy."
}

variable "environment_per_consortia" {
  type = "string"
  default = 1
  description = "The number of environments per consortia."
}

variable "nodes_per_environment" {
  type = "string"
  default = 3
  description = "The number of nodes to deploy per environment."
}

/*
This represents a Consortia. Give it a name and a description.
"mode" can be set to "single-org" or ...
*/
resource "kaleido_consortium" "mine" {
  count = "${var.consortia_count}"
  name = "Fulton's Cool Consortium${count.index}"
  description = "Toot Toot"
  mode = "single-org"
}

/*
This creates a membership, you can give it any "org_name" you like, but it has
to be linked to a Consortium via the Consortium resource "id".
*/
resource "kaleido_membership" "kaleido" {
  count = "${var.consortia_count}"
  consortium_id = "${element(kaleido_consortium.mine.*.id, count.index)}"
  org_name = "Fulton"
}

/*
Creates environments in Consortia.
*/
resource "kaleido_environment" "myEnv" {
  count = "${var.consortia_count * var.environment_per_consortia}"
  consortium_id = "${element(kaleido_consortium.mine.*.id, count.index % var.consortia_count)}"
  name = "Environment ${count.index}"
  description = "Pretty Cool"
  env_type = "${element(var.env_types, count.index % length(var.env_types))}"
  consensus_type = "${element(var.quorum_consensus, count.index % length(var.quorum_consensus))}"
}

/*
Creates a node in each environment, must be linked to a consortium, environment, and membership.
*/
resource "kaleido_node" "myNode" {
  count = "${var.nodes_per_environment * var.environment_per_consortia * var.consortia_count}" 
  consortium_id = "${element(kaleido_consortium.mine.*.id, count.index % var.consortia_count)}"
  environment_id = "${element(kaleido_environment.myEnv.*.id, count.index % length(kaleido_environment.myEnv.*.id))}"
  membership_id = "${element(kaleido_membership.kaleido.*.id, count.index % var.consortia_count)}"
  name = "node${count.index}"
}

/*
Creates an appkey for the "kaleido_membership" resource in
every environment.
*/
resource "kaleido_app_key" "appkey" {
  count = "${var.environment_per_consortia * var.consortia_count}"
  consortium_id = "${element(kaleido_consortium.mine.*.id, count.index % var.consortia_count)}"
  environment_id = "${element(kaleido_environment.myEnv.*.id, count.index % length(kaleido_environment.myEnv.*.id))}"
  membership_id = "${element(kaleido_membership.kaleido.*.id, count.index % var.consortia_count)}"
}
