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

func resourceDestination() *schema.Resource {
	return &schema.Resource{
		Create: resourceDestinationCreate,
		Read:   resourceDestinationRead,
		Update: resourceDestinationCreate, /* upsert/PUT semantics */
		Delete: resourceDestinationDelete,
		Schema: map[string]*schema.Schema{
			"service_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"kaleido_managed": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"consortium_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"membership_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"idregistry_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"auto_verify_membership": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
		},
	}
}

func resourceDestinationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	destination := kaleido.NewDestination(d.Get("name").(string))
	destination.KaleidoManaged = d.Get("kaleido_managed").(bool)
	consortiumID := d.Get("consortium_id").(string)
	membershipID := d.Get("membership_id").(string)
	serviceType := d.Get("service_type").(string)
	serviceID := d.Get("service_id").(string)
	idregistryID := d.Get("idregistry_id").(string)

	var membership kaleido.Membership
	res, err := client.GetMembership(consortiumID, membershipID, &membership)
	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to get membership %s in consortium %s with status %d: %s"
		return fmt.Errorf(msg, membershipID, consortiumID, status, res.String())
	}

	if d.Get("auto_verify_membership").(bool) {
		if membership.VerificationProof == "" {
			res, err = client.CreateMembershipVerification(consortiumID, membershipID, &kaleido.MembershipVerification{
				TestCertificate: true,
			})
			if err != nil {
				return err
			}
			status = res.StatusCode()
			if status != 200 {
				msg := "Failed to auto create self-signed membership verification proof for membership %s in consortium %s with status %d: %s"
				return fmt.Errorf(msg, membershipID, consortiumID, status, res.String())
			}
		}

		res, err = client.RegisterMembershipIdentity(idregistryID, membershipID)
		if err != nil {
			return err
		}
		status = res.StatusCode()
		if status != 200 && status != 409 /* already registered */ {
			msg := "Failed to auto register membership verification for membership %s in consortium %s using idregistry %s with status %d: %s"
			return fmt.Errorf(msg, membershipID, consortiumID, idregistryID, status, res.String())
		}
	}

	res, err = client.CreateDestination(serviceType, serviceID, &destination)

	if err != nil {
		return err
	}

	status = res.StatusCode()
	if status != 200 {
		msg := "Failed to create destination %s in %s service %s for membership %s with status %d: %s"
		return fmt.Errorf(msg, destination.Name, serviceType, serviceID, membershipID, status, res.String())
	}

	var destinations []kaleido.Destination
	res, err = client.ListDestinations(serviceType, serviceID, &destinations)

	if err != nil {
		return err
	}

	status = res.StatusCode()
	if status != 200 {
		msg := "Failed to query newly created destination %s in %s service %s for membership %s with status %d: %s"
		return fmt.Errorf(msg, destination.Name, serviceType, serviceID, membershipID, status, res.String())
	}

	var createdDest *kaleido.Destination
	for _, d := range destinations {
		if d.Name == destination.Name {
			createdDest = &d
			break
		}
	}

	if createdDest == nil {
		msg := "Failed to find newly created destination %s in %s service %s for membership %s"
		return fmt.Errorf(msg, destination.Name, serviceType, serviceID, membershipID)
	}

	d.SetId(createdDest.URI)

	return nil
}

func resourceDestinationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	serviceType := d.Get("service_type").(string)
	serviceID := d.Get("service_id").(string)
	destName := d.Get("name").(string)
	destURI := d.Id()

	var destinations []kaleido.Destination
	res, err := client.ListDestinations(serviceType, serviceID, &destinations)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Failed to list destinations in %s service %s with status %d: %s"
		return fmt.Errorf(msg, destName, serviceID, status, res.String())
	}

	var destination *kaleido.Destination
	for _, d := range destinations {
		if d.URI == destURI {
			destination = &d
			break
		}
	}

	if destination == nil {
		msg := "Failed to find destination %s in %s service %s"
		return fmt.Errorf(msg, destination.URI, serviceType, serviceID)
	}

	return nil
}

func resourceDestinationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	serviceType := d.Get("service_type").(string)
	serviceID := d.Get("service_id").(string)
	destName := d.Get("name").(string)

	res, err := client.DeleteDestination(serviceType, serviceID, destName)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 204 {
		msg := "Failed to delete destination %s in %s service %s with status %d: %s"
		return fmt.Errorf(msg, destName, serviceType, serviceID, status, res.String())
	}

	d.SetId("")
	return nil
}
