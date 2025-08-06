package platform

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var genesisStep = `
data "kaleido_platform_besu_qbft_network_genesis" "besu_qbft_network_genesis" {
	chain_id = 3333
	validator_keys = ["0x1234567890123456789012345678901234567890", "0xabcd09192d6eef99cc1234567890123456789012"]
}
`

func TestBesuQBFTNetworkGenesisDatasource(t *testing.T) {
	besuQBFTNetworkGenesisDatasource := "data.kaleido_platform_besu_qbft_network_genesis.besu_qbft_network_genesis"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: genesisStep,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(besuQBFTNetworkGenesisDatasource, "genesis", "{\"alloc\":{},\"coinbase\":\"0x0000000000000000000000000000000000000000\",\"config\":{\"chainId\":3333,\"berlinBlock\":0,\"contractSizeLimit\":98304,\"shanghaiTime\":0,\"zeroBaseFee\":true,\"qbft\":{\"blockperiodseconds\":5,\"epochlength\":30000,\"requesttimeoutseconds\":10}},\"difficulty\":\"0x1\",\"extraData\":\"0xf84fa00000000000000000000000000000000000000000000000000000000000000000ea94123456789012345678901234567890123456789094abcd09192d6eef99cc1234567890123456789012c080c0\",\"gasLimit\":\"0x2fefd800\",\"mixHash\":\"0x63746963616c2062797a616e74696e65206661756c7420746f6c6572616e6365\"}"),
				),
			},
		},
	})
}
