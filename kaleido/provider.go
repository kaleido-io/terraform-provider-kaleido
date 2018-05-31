package kaleido

import (
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
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
			"kaleido_consortium":  resourceConsortium(),
			"kaleido_environment": resourceEnvironment(),
			"kaleido_membership":  resourceMembership(),
			"kaleido_node":        resourceNode(),
			"kaleido_app_key":     resourceAppKey(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	api := d.Get("api").(string)
	apiKey := d.Get("api_key").(string)
	client := kaleido.NewClient(api, apiKey)
	return client, nil
}
