// A simple example of a platform user
resource "kaleido_platform_user" "user" {
  name  = "preferred-username"
  email = "user@example.com"
  sub   = "user-subject-identifier"
}

// A simple example of a platform user with admin privileges
resource "kaleido_platform_user" "admin_user" {
  name  = "preferred-username"
  email = "user@example.com"
  sub   = "user-subject-identifier"
  is_admin = true
}

// See 'kaleido_platform_group_membership' and 'kaleido_platform_group' for more examples of how to grant user's least privilege access to resources
