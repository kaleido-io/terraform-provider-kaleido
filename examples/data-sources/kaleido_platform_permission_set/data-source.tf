# A built-in set by name, to give it in a grant
data "kaleido_platform_permission_set" "full_access" {
  name = "full-access"
}

# Everything in one service, for an application
resource "kaleido_platform_grant" "ci_km1" {
  name              = "ci-km1"
  application_id    = kaleido_platform_application.ci.id
  permission_set_id = data.kaleido_platform_permission_set.full_access.id
  params_json       = jsonencode({ service = kaleido_platform_service.km1.id })
}
