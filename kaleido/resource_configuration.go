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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceConfigurationCreate,
		Read:   resourceConfigurationRead,
		Update: resourceConfigurationUpdate,
		Delete: resourceConfigurationDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"consortium_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"environment_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"membership_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"details": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"details_json": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceConfigurationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	membershipID := d.Get("membership_id").(string)
	configurationType := d.Get("type").(string)
	detailsMap := d.Get("details").(map[string]interface{})
	detailsJSON := d.Get("details_json").(string)
	if detailsJSON != "" {
		if err := json.Unmarshal([]byte(detailsJSON), &detailsMap); err != nil {
			msg := "Could not parse details_json of %s %s in consortium %s in environment %s: %s"
			return fmt.Errorf(msg, d.Get("type"), d.Get("name"), consortiumID, environmentID, err)
		}
	}
	details := duplicateDetails(detailsMap)
	configuration := kaleido.NewConfiguration(d.Get("name").(string), membershipID, configurationType, details)

	res, err := client.CreateConfiguration(consortiumID, environmentID, &configuration)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Could not create configuration %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, configuration.Type, consortiumID, environmentID, status, res.String())
	}

	res, err = client.GetConfiguration(consortiumID, environmentID, configuration.ID, &configuration)

	if err != nil {
		return err
	}

	statusCode := res.StatusCode()
	if statusCode != 200 {
		return fmt.Errorf("Fetching configuration %s state failed: %d", configuration.ID, statusCode)
	}

	d.SetId(configuration.ID)

	return nil
}

func resourceConfigurationUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	details := duplicateDetails(d.Get("details").(map[string]interface{}))
	configuration := kaleido.NewConfiguration(d.Get("name").(string), "", "", details)
	configID := d.Id()

	res, err := client.UpdateConfiguration(consortiumID, environmentID, configID, &configuration)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Could not update configuration %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, configID, consortiumID, environmentID, status, res.String())
	}

	return nil
}

func resourceConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	configurationID := d.Id()

	var configuration kaleido.Configuration
	res, err := client.GetConfiguration(consortiumID, environmentID, configurationID, &configuration)

	if err != nil {
		return err
	}

	status := res.StatusCode()

	if status == 404 {
		d.SetId("")
		return nil
	}
	if status != 200 {
		msg := "Could not find configuration %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, configurationID, consortiumID, environmentID, status, res.String())
	}

	d.Set("name", configuration.Name)
	d.Set("type", configuration.Type)
	return nil
}

func resourceConfigurationDelete(d *schema.ResourceData, meta interface{}) error {

	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	configurationID := d.Id()

	res, err := client.DeleteConfiguration(consortiumID, environmentID, configurationID)

	if err != nil {
		return err
	}

	statusCode := res.StatusCode()
	if statusCode != 202 && statusCode != 204 {
		msg := "Failed to delete configuration %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, environmentID, consortiumID, statusCode, res.String())
	}

	d.SetId("")

	return nil
}
