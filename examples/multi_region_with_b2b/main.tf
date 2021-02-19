/*
This creates suite of environments using all available
environment types and consensus methods.
*/

provider "kaleido" {
  api = var.kaleido_api_url
  api_key = var.kaleido_api_key
}

/*
This represents a Consortia. Give italeido_environment a name and a description.
"mode" can be set to "single-org" or ...
*/
resource "kaleido_consortium" "consortium" {
  name = var.consortium_name
  description = var.network_description
  shared_deployment = true
}

/*
This creates a membership for each node
*/
resource "kaleido_membership" "member" {
  count = var.member_count
  consortium_id = kaleido_consortium.consortium.id
  org_name = "Org ${count.index + 1}"
}

/*
This whitelists a region into the consortium for deployment (automatically re-uses an existing one if it exists)
*/
resource "kaleido_czone" "allowed_region" {
  consortium_id = kaleido_consortium.consortium.id
  cloud = var.cloud
  region = var.region
}

/*
Create an environment
*/
resource "kaleido_environment" "env" {
  consortium_id = kaleido_consortium.consortium.id
  multi_region = true
  name = var.env_name
  env_type = var.provider_type
  consensus_type = var.consensus
  description = var.env_description
  block_period = var.block_period
  shared_deployment = true
}

/*
This creates the deployment zone for your environment (automatically re-uses an existing one if it exists)
*/
resource "kaleido_ezone" "deployment_zone" {
  consortium_id = kaleido_consortium.consortium.id
  environment_id = kaleido_environment.env.id
  cloud = var.cloud
  region = var.region
}

/*
Create nodes
*/
resource "kaleido_node" "node" {
  count = var.member_count
  consortium_id = kaleido_consortium.consortium.id
  environment_id = kaleido_environment.env.id
  membership_id = element(kaleido_membership.member.*.id, count.index)
  zone_id = kaleido_ezone.deployment_zone.id
  name = "Node ${count.index + 1}"
  size = var.node_size
}

/*
Create idregistry
*/
resource "kaleido_service" "idregistry" {
  consortium_id = kaleido_consortium.consortium.id
  environment_id = kaleido_environment.env.id
  membership_id = element(kaleido_membership.member.*.id, 0)
  name = "On-chain registry"
  service_type = "idregistry"
  depends_on = [kaleido_node.node]
  shared_deployment = true
}

/*
Create app2app
*/
resource "kaleido_service" "app2app" {
  count = var.member_count
  consortium_id = kaleido_consortium.consortium.id
  environment_id = kaleido_environment.env.id
  membership_id = element(kaleido_membership.member.*.id, count.index)
  name = "App2app ${count.index + 1}"
  service_type = "app2app"
  size = "small" 
}

/*
Create app2app
*/
resource "kaleido_service" "docstore" {
  count = var.member_count
  consortium_id = kaleido_consortium.consortium.id
  environment_id = kaleido_environment.env.id
  membership_id = element(kaleido_membership.member.*.id, count.index)
  name = "Document Store ${count.index + 1}"
  service_type = "documentstore"
  size = "small" 
}

/*
Create app2app dest1
*/
resource "kaleido_destination" "a2a_dest1" {
  count = var.member_count
  consortium_id = kaleido_consortium.consortium.id
  membership_id = element(kaleido_membership.member.*.id, count.index)
  service_type = "app2app"
  service_id = element(kaleido_service.app2app.*.id, count.index)
  name = "a2a_dest1"
  kaleido_managed = true
  auto_verify_membership = true
  idregistry_id = kaleido_service.idregistry.id
  depends_on = [kaleido_service.app2app]
}

/*
Create app2app dest2
*/
resource "kaleido_destination" "a2a_dest2" {
  count = var.member_count
  consortium_id = kaleido_consortium.consortium.id
  membership_id = element(kaleido_membership.member.*.id, count.index)
  service_type = "app2app"
  service_id = element(kaleido_service.app2app.*.id, count.index)
  name = "a2a_dest2"
  kaleido_managed = true
  auto_verify_membership = true
  idregistry_id = kaleido_service.idregistry.id
  depends_on = [kaleido_service.app2app]
}

/*
Create docstore dest1
*/
resource "kaleido_destination" "docstore_dest1" {
  count = var.member_count
  consortium_id = kaleido_consortium.consortium.id
  membership_id = element(kaleido_membership.member.*.id, count.index)
  service_type = "documentstore"
  service_id = element(kaleido_service.docstore.*.id, count.index)
  name = "docstore_dest1"
  kaleido_managed = true
  auto_verify_membership = true
  idregistry_id = kaleido_service.idregistry.id
  depends_on = [kaleido_service.docstore]
}

/*
Create docstore dest2
*/
resource "kaleido_destination" "docstore_dest2" {
  count = var.member_count
  consortium_id = kaleido_consortium.consortium.id
  membership_id = element(kaleido_membership.member.*.id, count.index)
  service_type = "documentstore"
  service_id = element(kaleido_service.docstore.*.id, count.index)
  name = "docstore_dest2"
  kaleido_managed = true
  auto_verify_membership = true
  idregistry_id = kaleido_service.idregistry.id
  depends_on = [kaleido_service.docstore]
}
