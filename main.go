package main

import (
	"github.com/ConsenSys/photic-terraform-provider/photic"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return photic.Provider()
		},
	})
}
