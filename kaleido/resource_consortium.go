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
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceConsortium() *schema.Resource {
	return &schema.Resource{
		Create: resourceConsortiumCreate,
		Read:   resourceConsortiumRead,
		Update: resourceConsortiumUpdate,
		Delete: resourceConsortiumDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"shared_deployment": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "The decentralized nature of Kaleido means a consortium might be shared with other accounts. When true only create if name does not exist, and delete becomes a no-op.",
			},
		},
	}
}

func resourceConsortiumCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortium := kaleido.NewConsortium(
		d.Get("name").(string),
		d.Get("description").(string),
	)

	if d.Get("shared_deployment").(bool) {
		var consortia []kaleido.Consortium
		res, err := client.ListConsortium(&consortia)
		if err != nil {
			return err
		}
		if res.StatusCode() != 200 {
			return fmt.Errorf("Failed to list existing consortia with status %d: %s", res.StatusCode(), res.String())
		}
		for _, c := range consortia {
			if c.Name == consortium.Name && !strings.Contains(c.State, "delete") {
				// Already exists, just re-use
				d.SetId(c.ID)
				return resourceConsortiumRead(d, meta)
			}
		}
	}

	res, err := client.CreateConsortium(&consortium)
	if err != nil {
		return err
	}
	status := res.StatusCode()
	if status != 201 {
		return fmt.Errorf("Failed to create consortium with status %d", status)
	}

	d.SetId(consortium.ID)

	return nil
}

func resourceConsortiumUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortium := kaleido.NewConsortium(
		d.Get("name").(string),
		d.Get("description").(string),
	)
	consortiumID := d.Id()

	res, err := client.UpdateConsortium(consortiumID, &consortium)
	if err != nil {
		return err
	}
	status := res.StatusCode()
	if status != 200 {
		return fmt.Errorf("Failed to update consortium %s with status: %d", consortiumID, status)
	}

	return nil
}

func resourceConsortiumRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	var consortium kaleido.Consortium
	res, err := client.GetConsortium(d.Id(), &consortium)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		if status == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Failed to read consortium with id %s status was: %d, error: %s", d.Id(), status, res.String())
	}
	d.Set("name", consortium.Name)
	d.Set("description", consortium.Description)
	return nil
}

func resourceConsortiumDelete(d *schema.ResourceData, meta interface{}) error {
	if d.Get("shared_deployment").(bool) {
		// Cannot safely delete if this is shared with other terraform deployments
		d.SetId("")
		return nil
	}

	client := meta.(kaleido.KaleidoClient)
	res, err := client.DeleteConsortium(d.Id())

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 202 {
		return fmt.Errorf("failed to delete consortium with id %s status was %d, error: %s", d.Id(), status, res.String())
	}
	return nil
}
