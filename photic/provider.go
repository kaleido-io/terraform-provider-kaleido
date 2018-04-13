package photic

import (
	photic "github.com/Consensys/photic-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("KALEIDO_API", nil),
			},
			"api_key": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("KALEIDO_API_KEY", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"photic_consortium":  resourceConsortium(),
			"photic_environment": resourceEnvironment(),
			"photic_membership":  resourceMembership(),
			"photic_node":        resourceNode(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	api := d.Get("api").(string)
	apiKey := d.Get("api_key").(string)
	client := photic.NewClient(api, apiKey)
	return client, nil
}
