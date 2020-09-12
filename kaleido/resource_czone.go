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

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceCZone() *schema.Resource {
	return &schema.Resource{
		Create: resourceCZoneCreate,
		Read:   resourceCZoneRead,
		Delete: resourceCZoneDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
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

	res, err := client.CreateCZone(consortiumID, &czone)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Could not create czone %s in consortium %s in environment %s, status was: %d, error: %s"
		return fmt.Errorf(msg, czone.ID, consortiumID, status, res.String())
	}

	err = resource.Retry(d.Timeout("Create"), func() *resource.RetryError {
		res, retryErr := client.GetCZone(consortiumID, czone.ID, &czone)

		if retryErr != nil {
			return resource.NonRetryableError(retryErr)
		}

		statusCode := res.StatusCode()
		if statusCode != 200 {
			msg := fmt.Errorf("Fetching czone %s state failed: %d", czone.ID, statusCode)
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
		msg := "Could not find czone %s in consortium %s in environment %s, status: %d"
		return fmt.Errorf(msg, czoneID, consortiumID, status)
	}

	d.Set("name", czone.Name)
	return nil
}

func resourceCZoneDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
