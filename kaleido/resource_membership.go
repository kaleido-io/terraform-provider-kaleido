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
