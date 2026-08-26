resource "kaleido_platform_environment" "env" {
  name            = "environment_name"
  version         = "26.1.0"
  update_strategy = "manual"
}

# The versions this environment could be upgraded to, newest first
data "kaleido_platform_environment_versions" "env" {
  environment = kaleido_platform_environment.env.id
}

output "environment_upgrade_available" {
  value = data.kaleido_platform_environment_versions.env.upgrade_available
}

# Null when the environment is already running the newest version available to it
output "environment_latest_version" {
  value = data.kaleido_platform_environment_versions.env.latest_version
}
