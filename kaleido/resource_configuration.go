// Copyright 2018 Kaleido, a ConsenSys business

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
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceConfigurationCreate,
		Read:   resourceConfigurationRead,
		Delete: resourceConfigurationDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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
				ForceNew: true,
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
	details := d.Get("details").(map[string]interface{})
	configuration := kaleido.NewConfiguration(d.Get("name").(string), membershipID, configurationType, details)

	res, err := client.CreateConfiguration(consortiumID, environmentID, &configuration)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Could not create configuration %s in consortium %s in environment %s, status was: %d"
		return fmt.Errorf(msg, configuration.ID, consortiumID, environmentID, status)
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
		msg := "Could not find configuration %s in consortium %s in environment %s, status: %d"
		return fmt.Errorf(msg, configurationID, consortiumID, environmentID, status)
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
		msg := "Failed to delete configuration %s in consortium %s in environment %s, status: %d"
		return fmt.Errorf(msg, environmentID, consortiumID, statusCode)
	}

	d.SetId("")

	return nil
}
