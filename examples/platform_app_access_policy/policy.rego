package platform_permission

import rego.v1

default allow := false

# Only allow access to specific environments by name
is_valid_env(env) := env in ["production", "staging"]

allow if {
    is_valid_env(input.environment.name)
}
