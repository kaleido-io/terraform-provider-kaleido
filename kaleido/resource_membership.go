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

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceMembership() *schema.Resource {
	return &schema.Resource{
		Create: resourceMembershipCreate,
		Read:   resourceMembershipRead,
		Update: resourceMembershipUpdate,
		Delete: resourceMembershipDelete,
		Schema: map[string]*schema.Schema{
			"consortium_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"org_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceMembershipCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	membership := kaleido.NewMembership(d.Get("org_name").(string))
	consortiumID := d.Get("consortium_id").(string)

	res, err := client.CreateMembership(consortiumID, &membership)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Failed to create membership %s in consortium %s with status %d"
		return fmt.Errorf(msg, membership.OrgName, consortiumID, status)
	}

	d.SetId(membership.ID)
	return nil
}

func resourceMembershipUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	membership := kaleido.NewMembership(d.Get("org_name").(string))
	consortiumID := d.Get("consortium_id").(string)
	membershipID := d.Id()

	res, err := client.UpdateMembership(consortiumID, membershipID, &membership)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to update membership %s for %s in consortium %s with status %d"
		return fmt.Errorf(msg, membershipID, membership.OrgName, consortiumID, status)
	}

	return nil
}

func resourceMembershipRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)

	var membership kaleido.Membership
	res, err := client.GetMembership(consortiumID, d.Id(), &membership)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to find membership %s in consortium %s with status %d"
		return fmt.Errorf(msg, membership.OrgName, consortiumID, status)
	}

	d.Set("org_name", membership.OrgName)
	return nil
}

func resourceMembershipDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	membershipID := d.Id()

	err := resource.Retry(d.Timeout("Delete"), func() *resource.RetryError {
		res, retryErr := client.DeleteMembership(consortiumID, membershipID)

		if retryErr != nil {
			return resource.NonRetryableError(retryErr)
		}

		statusCode := res.StatusCode()
		if statusCode >= 500 {
			msg := fmt.Errorf("deletion of membership %s failed: %d", membershipID, statusCode)
			return resource.NonRetryableError(msg)
		} else if statusCode != 204 {
			msg := "Failed to delete membership %s in consortium %s with status: %d"
			return resource.RetryableError(fmt.Errorf(msg, membershipID, consortiumID, statusCode))
		}

		return nil
	})

	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
