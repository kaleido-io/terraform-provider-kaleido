resource "kaleido_platform_ars_namespace" "files" {
  environment       = kaleido_platform_environment.env_0.id
  service           = kaleido_platform_service.ars_0.id
  name              = "demo"
  artifact_family   = "file"
  auto_create_repos = false
}

# With auto_create_repos = false every repository must be created before artifacts are pushed
resource "kaleido_platform_ars_repository" "routing_table" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.ars_0.id
  namespace   = kaleido_platform_ars_namespace.files.name
  name        = "resources/routing-table.yaml"
  description = "Routing tables"
}

resource "kaleido_platform_ars_file_artifact" "routing_table" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.ars_0.id
  namespace   = kaleido_platform_ars_namespace.files.name
  name        = kaleido_platform_ars_repository.routing_table.name
  file_path   = "${path.module}/files/route-table.yaml"
  type        = "yaml"
  tag         = "rel-2026-10.001"
}
