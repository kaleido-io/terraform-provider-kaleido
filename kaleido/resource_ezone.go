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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceEZone() *schema.Resource {
	return &schema.Resource{
		Create: resourceEZoneCreate,
		Read:   resourceEZoneRead,
		Update: resourceEZoneUpdate,
		Delete: resourceEZoneDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"cloud": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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

func resourceEZoneCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	region := d.Get("region").(string)
	cloud := d.Get("cloud").(string)
	ezone := kaleido.NewEZone(d.Get("name").(string), region, cloud)

	var existing []kaleido.EZone
	res, err := client.ListEZones(consortiumID, environmentID, &existing)
	if err != nil {
		return err
	}
	if res.StatusCode() != 200 {
		return fmt.Errorf("Failed to list existing environment zones with status %d: %s", res.StatusCode(), res.String())
	}
	for _, e := range existing {
		if e.Cloud == cloud && e.Region == region {
			// Already exists, just re-use
			d.SetId(e.ID)
			return resourceEZoneRead(d, meta)
		}
	}

	res, err = client.CreateEZone(consortiumID, environmentID, &ezone)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Could not create ezone in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, consortiumID, environmentID, status, res.String())
	}

	err = resource.Retry(d.Timeout("Create"), func() *resource.RetryError {
		res, retryErr := client.GetEZone(consortiumID, environmentID, ezone.ID, &ezone)

		if retryErr != nil {
			return resource.NonRetryableError(retryErr)
		}

		statusCode := res.StatusCode()
		if statusCode != 200 {
			msg := fmt.Errorf("Fetching ezone %s state failed with status %d: %s", ezone.ID, statusCode, res.String())
			return resource.NonRetryableError(msg)
		}

		return nil
	})

	if err != nil {
		return err
	}

	d.SetId(ezone.ID)

	return nil
}

func resourceEZoneUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	ezoneID := d.Id()
	ezone := kaleido.NewEZone(d.Get("name").(string), "", "")

	res, err := client.UpdateEZone(consortiumID, environmentID, ezoneID, &ezone)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Could not update ezone %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, ezoneID, consortiumID, environmentID, status, res.String())
	}

	return nil
}

func resourceEZoneRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	ezoneID := d.Id()

	var ezone kaleido.EZone
	res, err := client.GetEZone(consortiumID, environmentID, ezoneID, &ezone)

	if err != nil {
		return err
	}

	status := res.StatusCode()

	if status == 404 {
		d.SetId("")
		return nil
	}
	if status != 200 {
		msg := "Could not find ezone %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, ezoneID, consortiumID, environmentID, status, res.String())
	}

	d.Set("name", ezone.Name)
	return nil
}

func resourceEZoneDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
