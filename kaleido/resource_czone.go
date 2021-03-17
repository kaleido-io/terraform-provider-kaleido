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

func resourceCZone() *schema.Resource {
	return &schema.Resource{
		Create: resourceCZoneCreate,
		Read:   resourceCZoneRead,
		Update: resourceCZoneUpdate,
		Delete: resourceCZoneDelete,
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

func resourceCZoneCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	region := d.Get("region").(string)
	cloud := d.Get("cloud").(string)
	czone := kaleido.NewCZone(d.Get("name").(string), region, cloud)

	var existing []kaleido.CZone
	res, err := client.ListCZones(consortiumID, &existing)
	if err != nil {
		return err
	}
	if res.StatusCode() != 200 {
		return fmt.Errorf("Failed to list existing consortia zones with status %d: %s", res.StatusCode(), res.String())
	}
	for _, e := range existing {
		if e.Cloud == cloud && e.Region == region {
			// Already exists, just re-use
			d.SetId(e.ID)
			return resourceCZoneRead(d, meta)
		}
	}

	res, err = client.CreateCZone(consortiumID, &czone)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Could not create czone in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, consortiumID, status, res.String())
	}

	err = resource.Retry(d.Timeout("Create"), func() *resource.RetryError {
		res, retryErr := client.GetCZone(consortiumID, czone.ID, &czone)

		if retryErr != nil {
			return resource.NonRetryableError(retryErr)
		}

		statusCode := res.StatusCode()
		if statusCode != 200 {
			msg := fmt.Errorf("Fetching czone %s state failed with status %d: %s", czone.ID, statusCode, res.String())
			return resource.NonRetryableError(msg)
		}

		return nil
	})

	if err != nil {
		return err
	}

	d.SetId(czone.ID)

	return nil
}

func resourceCZoneUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	czoneID := d.Id()
	czone := kaleido.NewCZone(d.Get("name").(string), "", "")

	res, err := client.UpdateCZone(consortiumID, czoneID, &czone)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Could not update czone %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, czoneID, consortiumID, status, res.String())
	}

	return nil
}

func resourceCZoneRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	czoneID := d.Id()

	var czone kaleido.CZone
	res, err := client.GetCZone(consortiumID, czoneID, &czone)

	if err != nil {
		return err
	}

	status := res.StatusCode()

	if status == 404 {
		d.SetId("")
		return nil
	}
	if status != 200 {
		msg := "Could not find czone %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, czoneID, consortiumID, status, res.String())
	}

	d.Set("name", czone.Name)
	return nil
}

func resourceCZoneDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
