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
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourcePrivateStackBridge() *schema.Resource {
	return &schema.Resource{
		Read: resourcePrivateStackBridgeRead,

		Schema: map[string]*schema.Schema{
			"consortium_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"environment_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"service_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"config_json": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePrivateStackBridgeRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(kaleido.KaleidoClient)

	var conf map[string]interface{}
	res, err := client.GetPrivateStackBridgeConfig(d.Get("consortium_id").(string), d.Get("environment_id").(string), d.Get("service_id").(string), &conf)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		return fmt.Errorf("Failed to read config with id %s status was: %d, error: %s", d.Id(), status, res.String())
	}
	d.SetId(d.Get("service_id").(string))
	confstr, _ := json.MarshalIndent(conf, "", "  ")
	d.Set("config_json", string(confstr))
	return nil
}
