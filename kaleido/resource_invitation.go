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

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceInvitation() *schema.Resource {
	return &schema.Resource{
		Create: resourceInvitationCreate,
		Read:   resourceInvitationRead,
		Update: resourceInvitationUpdate,
		Delete: resourceInvitationDelete,
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
			"email": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceInvitationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	invitation := kaleido.NewInvitation(d.Get("org_name").(string), d.Get("email").(string))
	consortiumID := d.Get("consortium_id").(string)

	res, err := client.CreateInvitation(consortiumID, &invitation)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Failed to create invitation %s in consortium %s with status %d"
		return fmt.Errorf(msg, invitation.OrgName, consortiumID, status)
	}

	d.SetId(invitation.ID)
	return nil
}

func resourceInvitationUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	invitation := kaleido.NewInvitation(d.Get("org_name").(string), d.Get("email").(string))
	consortiumID := d.Get("consortium_id").(string)
	inviteID := d.Id()

	res, err := client.UpdateInvitation(consortiumID, inviteID, &invitation)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to update invitation %s for %s in consortium %s with status %d"
		return fmt.Errorf(msg, inviteID, invitation.OrgName, consortiumID, status)
	}

	return nil
}

func resourceInvitationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)

	var invitation kaleido.Invitation
	res, err := client.GetInvitation(consortiumID, d.Id(), &invitation)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to find invitation %s in consortium %s with status %d"
		return fmt.Errorf(msg, invitation.OrgName, consortiumID, status)
	}

	d.Set("org_name", invitation.OrgName)
	d.Set("email", invitation.Email)
	return nil
}

func resourceInvitationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	invitationID := d.Id()

	res, err := client.DeleteInvitation(consortiumID, invitationID)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 204 {
		msg := "Failed to delete invitation %s in consortium %s with status: %d"
		return fmt.Errorf(msg, invitationID, consortiumID, status)
	}

	d.SetId("")
	return nil
}
