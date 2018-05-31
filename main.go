package main

import (
	"github.com/kaleido-io/terraform-provider-kaleido/kaleido"

	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return kaleido.Provider()
		},
	})
}
