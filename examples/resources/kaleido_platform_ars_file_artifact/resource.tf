resource "kaleido_platform_ars_namespace" "files" {
  environment     = kaleido_platform_environment.env_0.id
  service         = kaleido_platform_service.ars_0.id
  name            = "demo"
  artifact_family = "file"
}

# Recommended: content-addressed tagging. The tag is derived as
# "<version>-<first 8 hex chars of the file's SHA-256>", so any change to the
# local file uploads the artifact under a new tag.
resource "kaleido_platform_ars_file_artifact" "indexer_typescript" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.ars_0.id
  namespace   = kaleido_platform_ars_namespace.files.name
  name        = "tssdk_modules/indexer.ts"
  file_path   = "${path.module}/tssdk_modules/src/indexer.ts"
  type        = "json"
  version     = "v1.0.0" # the tag will be suffixed with a shortened checksum

  # Old versions are retained in the registry on upgrade by default; set this
  # to remove the previously tracked version once the new one uploads
  remove_old_versions = true
}

# Explicit tagging: the tag's existence is trusted and local file changes are
# NOT detected or re-uploaded. Tags are immutable server-side, so pushing to
# an existing tag with different content fails.
resource "kaleido_platform_ars_file_artifact" "release_notes" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.ars_0.id
  namespace   = kaleido_platform_ars_namespace.files.name
  name        = "resources/routing-table.yaml"
  file_path   = "${path.module}/files/route-table.yaml"
  type        = "yaml"
  tag         = "rel-2026-07.001"
}
