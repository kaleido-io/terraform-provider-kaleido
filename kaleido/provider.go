// Copyright © Kaleido, Inc. 2018, 2021

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package kaleido

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
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
			"kaleido_consortium":    resourceConsortium(),
			"kaleido_environment":   resourceEnvironment(),
			"kaleido_membership":    resourceMembership(),
			"kaleido_node":          resourceNode(),
			"kaleido_service":       resourceService(),
			"kaleido_app_creds":     resourceAppCreds(),
			"kaleido_invitation":    resourceInvitation(),
			"kaleido_czone":         resourceCZone(),
			"kaleido_ezone":         resourceEZone(),
			"kaleido_configuration": resourceConfiguration(),
			"kaleido_destination":   resourceDestination(),
		},
		ConfigureFunc: providerConfigure,
		DataSourcesMap: map[string]*schema.Resource{
			"kaleido_privatestack_bridge": resourcePrivateStackBridge(),
		},
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	api := d.Get("api").(string)
	apiKey := d.Get("api_key").(string)
	client := kaleido.NewClient(api, apiKey)
	return client, nil
}
