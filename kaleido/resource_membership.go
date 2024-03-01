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

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceMembership() resource.Resource {
	return &resource.Resource{
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
			"pre_existing": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "In a decentalized consortium memberships are driven by invitation, and will be pre-existing at the point of deploying infrastructure.",
			},
		},
	}
}

func resourceMembershipCreate(d *resource.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	orgName := d.Get("org_name").(string)
	membership := kaleido.NewMembership(orgName)
	consortiumID := d.Get("consortium_id").(string)

	if d.Get("pre_existing").(bool) {
		var memberships []kaleido.Membership
		res, err := client.ListMemberships(consortiumID, &memberships)
		if err != nil {
			return err
		}
		if res.StatusCode() != 200 {
			return fmt.Errorf("Failed to list existing memberships with status %d: %s", res.StatusCode(), res.String())
		}
		for _, e := range memberships {
			if e.OrgName == orgName {
				d.SetId(e.ID)
				return resourceMembershipRead(d, meta)
			}
		}
		msg := "pre_existing set and no existing membership found with org_name '%s'"
		return fmt.Errorf(msg, orgName)
	}

	res, err := client.CreateMembership(consortiumID, &membership)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Failed to create membership %s in consortium %s with status %d: %s"
		return fmt.Errorf(msg, membership.OrgName, consortiumID, status, res.String())
	}

	d.SetId(membership.ID)
	return nil
}

func resourceMembershipUpdate(d *resource.ResourceData, meta interface{}) error {
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
		msg := "Failed to update membership %s for %s in consortium %s with status %d: %s"
		return fmt.Errorf(msg, membershipID, membership.OrgName, consortiumID, status, res.String())
	}

	return nil
}

func resourceMembershipRead(d *resource.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)

	var membership kaleido.Membership
	res, err := client.GetMembership(consortiumID, d.Id(), &membership)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to find membership %s in consortium %s with status %d: %s"
		return fmt.Errorf(msg, membership.OrgName, consortiumID, status, res.String())
	}

	d.Set("org_name", membership.OrgName)
	return nil
}

func resourceMembershipDelete(d *resource.ResourceData, meta interface{}) error {
	if d.Get("pre_existing").(bool) {
		// Cannot safely delete if this is shared with other terraform deployments
		d.SetId("")
		return nil
	}

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
			msg := "Failed to delete membership %s in consortium %s with status %d: %s"
			return resource.RetryableError(fmt.Errorf(msg, membershipID, consortiumID, statusCode, res.String()))
		}

		return nil
	})

	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
