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

	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceMembership() *schema.Resource {
	return &schema.Resource{
		Create: resourceMembershipCreate,
		Read:   resourceMembershipRead,
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
				ForceNew: true,
			},
		},
	}
}

func resourceMembershipCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	membership := kaleido.NewMembership(d.Get("org_name").(string))
	consortiumId := d.Get("consortium_id").(string)

	res, err := client.CreateMembership(consortiumId, &membership)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Failed to create membership %s in consortium %s with status %d"
		return fmt.Errorf(msg, membership.OrgName, consortiumId, status)
	}

	d.SetId(membership.Id)
	return nil
}

func resourceMembershipRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)

	var membership kaleido.Membership
	res, err := client.GetMembership(consortiumId, d.Id(), &membership)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to find membership %s in consortium %s with status %d"
		return fmt.Errorf(msg, membership.OrgName, consortiumId, status)
	}

	d.Set("org_name", membership.OrgName)
	return nil
}

func resourceMembershipDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	membershipId := d.Id()

	res, err := client.DeleteMembership(consortiumId, membershipId)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 204 {
		msg := "Failed to delete membership %s in consortium %s with status: %d"
		return fmt.Errorf(msg, membershipId, consortiumId, status)
	}

	d.SetId("")
	return nil
}
