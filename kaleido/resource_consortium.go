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

	"github.com/hashicorp/terraform/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceConsortium() *schema.Resource {
	return &schema.Resource{
		Create: resourceConsortiumCreate,
		Read:   resourceConsortiumRead,
		Delete: resourceConsortiumDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceConsortiumCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortium := kaleido.NewConsortium(d.Get("name").(string),
		d.Get("description").(string))
	res, err := client.CreateConsortium(&consortium)
	if err != nil {
		return err
	}
	status := res.StatusCode()
	if status != 201 {
		return fmt.Errorf("Failed to create consortium with status: %d", status)
	}

	d.SetId(consortium.Id)

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
		return fmt.Errorf("Failed to read consortium with id %s status was: %d", d.Id(), status)
	}
	return nil
}

func resourceConsortiumDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	res, err := client.DeleteConsortium(d.Id())

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 202 {
		return fmt.Errorf("failed to delete consortium with id %s status was %d", d.Id(), status)
	}
	return nil
}
