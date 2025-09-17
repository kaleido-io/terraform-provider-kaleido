// A simple example of a platform user
resource "kaleido_platform_user" "user" {
  name  = "preferred-username"
  email = "user@example.com"
  sub   = "user-subject-identifier"
}
