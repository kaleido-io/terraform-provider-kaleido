resource "kaleido_platform_environment" "env" {
  name            = "environment_name"
  version         = "26.1.0"
  update_strategy = "manual"
}

# The versions this environment could be upgraded to, newest first
data "kaleido_platform_environment_versions" "env" {
  environment = kaleido_platform_environment.env.id
}

output "environment_latest_version" {
  value = data.kaleido_platform_environment_versions.env.latest_version
}

check "environment_up_to_date" {
  data "kaleido_platform_environment_versions" "upgrade" {
    environment = kaleido_platform_environment.env.id
  }

  assert {
    condition     = !data.kaleido_platform_environment_versions.upgrade.upgrade_available
    error_message = "Environment ${kaleido_platform_environment.env.name} can be upgraded to ${data.kaleido_platform_environment_versions.upgrade.latest_version}"
  }
}
