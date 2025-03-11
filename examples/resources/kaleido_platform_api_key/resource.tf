resource "kaleido_platform_api_key" "api_key" {
  name = "api_key"
  application_id = kaleido_platform_application.application.id
  no_expiry = true
}