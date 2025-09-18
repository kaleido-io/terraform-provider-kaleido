// A simple example of a platform account
resource "kaleido_platform_account" "simple_example" {
  name = "example"
  // This refers to an existing OIDC Client in the platform, configured to login users via a trusted identity provider
  oidc_client_id = "i:abcd1234"
  // This trusted first user will be added to the account on creation, and will have admin access after first login
  first_user_email = "user@example.com"
  // Results in example.${PLATFORM_DOMAIN} for the account FQDN
  hostnames = {
    "example" = []
  }
}

// A more complex example of a platform account with validation policy, user JIT, and bootstrap application
resource "kaleido_platform_account" "robust_example" {
  name             = "example"
  oidc_client_id   = "i:abcd1234"
  first_user_email = "user@example.com"
  hostnames = {
    "example" = []
  }

  // Users matching the validation policy will be automatically "imported" on first login, and added to the 'readers' group if one exists
  user_jit_enabled       = true
  user_jit_default_group = "readers"

  // This validation policy allows only users with either the admin or owner roles to access the account
  validation_policy = <<EOT
package example_validation
import future.keywords.in

default allow := false

is_valid_aud(aud) := aud == "expected_client_id"

is_valid_admin(roles) := roles == ["admin"]

is_valid_owner(roles) := roles == ["owner"]

allow {
	is_valid_aud(input.id_token.aud)
	is_valid_admin(input.id_token.roles)
}

allow {
	is_valid_aud(input.id_token.aud)
	is_valid_owner(input.id_token.roles)
}
EOT
}
