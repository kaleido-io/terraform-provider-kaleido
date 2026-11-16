# Read, resolve and sign with the keys labelled team=treasury in one keystore of a Key Manager,
# and read that keystore. The keystore is a param: each grant of the set names its own.
resource "kaleido_platform_permission_set" "key_signers" {
  name        = "treasury-key-signers"
  description = "Read, resolve and sign with the treasury keys in a keystore"
  permissions_json = jsonencode([
    {
      actions = ["read", "resolve", "sign"]
      resourceSelector = [
        { field = "resource.type", op = "eq", values = ["keymanager.key"] },
        { field = "resource.within.keystore.name", op = "eq", param = "keystore" },
        { field = "resource.labels.team", op = "eq", values = ["treasury"] },
      ]
    },
    {
      actions = ["read"]
      resourceSelector = [
        { field = "resource.type", op = "eq", values = ["keymanager.keystore"] },
        { field = "resource.name", op = "eq", param = "keystore" },
      ]
    },
  ])
  params_json = jsonencode([{ name = "keystore", description = "The keystore" }])
}
