
resource "kaleido_platform_runtime" "runtime1" {
    environment = "env1"
    type = "besu"
    name = "runtime1"
    config_json = jsonencode({
        "setting1": "value1"
    })
    log_level = "debug"
    size = "small"
}
