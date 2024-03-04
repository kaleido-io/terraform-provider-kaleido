// Copyright © Kaleido, Inc. 2024

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package platform

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

type Runtime struct {
	ResourceCommon
	Type                string                 `json:"type"`
	Name                string                 `json:"name"`
	Config              map[string]interface{} `json:"config"`
	LogLevel            string                 `json:"loglevel"`
	Size                string                 `json:"size"`
	EnvironmentMemberID string                 `json:"environmentMemberId"`
	//read only
	Status          string `json:"status"`
	Deleted         bool   `json:"deleted"`
	ExplicitStopped bool   `json:"stopped"`
}

func ResourceRuntime() resource.Resource {
	return &resource.Resource{
		Create: resourceRuntimeCreate,
		Read:   resourceRuntimeRead,
		Update: resourceRuntimeUpdate,
		Delete: resourceRuntimeDelete,
		Schema: withCommon(map[string]*schema.Schema{
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"environmentMemberId": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Required: false,
			},
			"loglevel": &schema.Schema{
				Type:     schema.TypeString,
				Required: false,
			},
			"size": &schema.Schema{
				Type:     schema.TypeString,
				Required: false,
			},
		}),
	}
}

func resourceRuntimeCreate(d *resource.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)

	res, err := createCommon(&consortium)
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

func resourceRuntimeUpdate(d *resource.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortium := kaleido.NewRuntime(
		d.Get("name").(string),
		d.Get("description").(string),
	)
	consortiumID := d.Id()

	res, err := client.UpdateRuntime(consortiumID, &consortium)
	if err != nil {
		return err
	}
	status := res.StatusCode()
	if status != 200 {
		return fmt.Errorf("Failed to update consortium %s with status: %d", consortiumID, status)
	}

	return nil
}

func resourceRuntimeRead(d *resource.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	var consortium kaleido.Runtime
	res, err := client.GetRuntime(d.Id(), &consortium)

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

func resourceRuntimeDelete(d *resource.ResourceData, meta interface{}) error {
	if d.Get("shared_deployment").(bool) {
		// Cannot safely delete if this is shared with other terraform deployments
		d.SetId("")
		return nil
	}

	client := meta.(kaleido.KaleidoClient)
	res, err := client.DeleteRuntime(d.Id())

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 202 {
		return fmt.Errorf("failed to delete consortium with id %s status was %d, error: %s", d.Id(), status, res.String())
	}
	return nil
}
