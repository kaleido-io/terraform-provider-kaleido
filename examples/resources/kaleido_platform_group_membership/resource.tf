// A simple example of a platform group membership for a user
resource "kaleido_platform_group_membership" "user_group_membership" {
  group_id = "g:abcd1234"
  user_id  = "u:1234abcd"
}
