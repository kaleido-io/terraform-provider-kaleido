resource "kaleido_platform_ars_namespace" "files" {
  environment     = kaleido_platform_environment.env_0.id
  service         = kaleido_platform_service.ars_0.id
  name            = "demo"
  artifact_family = "file"
}

# Recommended: content-addressed tagging. The tag is derived as
# "<version>-<first 8 hex chars of the file's SHA-256>", so any change to the
# local file replaces the artifact under a new tag.
resource "kaleido_platform_ars_file_artifact" "schema" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.ars_0.id
  namespace   = kaleido_platform_ars_namespace.files.name
  name        = "schemas/payments/pacs008.json"
  file_path   = "${path.module}/files/pacs008.json"
  type        = "json"
  version     = "v1.0.0"
}

# Explicit tagging: the tag's existence is trusted and local file changes are
# NOT detected or re-uploaded. Tags are immutable server-side, so creating
# against an existing tag with different content fails.
resource "kaleido_platform_ars_file_artifact" "release_notes" {
  environment = kaleido_platform_environment.env_0.id
  service     = kaleido_platform_service.ars_0.id
  namespace   = kaleido_platform_ars_namespace.files.name
  name        = "docs/release-notes.yaml"
  file_path   = "${path.module}/files/release-notes.yaml"
  type        = "yaml"
  tag         = "rel-2026-07"
}
