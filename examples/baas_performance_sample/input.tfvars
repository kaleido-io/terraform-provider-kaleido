kaleido_api_key = "<Platform API Key>"
consortium_name = "test-consortium"
env_name = "test-environment"
provider_type= "pantheon"
consensus = "ibft"
multi_region = "true"
node_count = 4
block_period = 5
node_size = "large"
protocol_config = {
      "restgw_max_inflight": 1000,
      "restgw_max_tx_wait_time": 60,
      "restgw_always_manage_nonce": true,
      "restgw_send_concurrency": 100,
      "restgw_attempt_gap_fill": true,
      "restgw_flush_frequency": 0,
      "restgw_flush_msgs": 0,
      "restgw_flush_bytes": 0,
    }

