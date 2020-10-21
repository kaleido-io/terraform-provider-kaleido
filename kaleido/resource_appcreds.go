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

	"github.com/hashicorp/terraform/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceAppCreds() *schema.Resource {
	return &schema.Resource{
		Create: resourceAppCredCreate,
		Read:   resourceAppCredRead,
		Delete: resourceAppCredDelete,
		Schema: map[string]*schema.Schema{
			"membership_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"password": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"auth_type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAppCredCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	envID := d.Get("environment_id").(string)
	membershipID := d.Get("membership_id").(string)
	name := d.Get("name").(string)

	var appKey kaleido.AppCreds
	if name == "" {
		appKey = kaleido.NewAppCreds(membershipID)
	} else {
		appKey = kaleido.NewAppCredsWithName(membershipID, name)
	}

	res, err := client.CreateAppCreds(consortiumID, envID, &appKey)

	if err != nil {
		return err
	}

	if res.StatusCode() != 201 {
		msg := "Could not create AppKey in consortium %s, in environment %s, with membership %s. Status: %d"
		return fmt.Errorf(msg, consortiumID, envID, membershipID, res.StatusCode())
	}

	d.SetId(appKey.ID)
	d.Set("username", appKey.Username)
	d.Set("password", appKey.Password)
	d.Set("auth_type", appKey.AuthType)

	return nil
}

func resourceAppCredRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	envID := d.Get("environment_id").(string)
	appKeyID := d.Id()

	var appKey kaleido.AppCreds
	res, err := client.GetAppCreds(consortiumID, envID, appKeyID, &appKey)

	if err != nil {
		return err
	}

	if res.StatusCode() != 200 {
		msg := "Could not fetch AppKey %s in consortium %s, in environment %s. Status: %d."
		return fmt.Errorf(msg, appKeyID, consortiumID, envID, res.StatusCode())
	}

	d.Set("auth_type", appKey.AuthType)
	return nil
}

func resourceAppCredDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	envID := d.Get("environment_id").(string)
	appKeyID := d.Id()

	res, err := client.DeleteAppCreds(consortiumID, envID, appKeyID)

	if err != nil {
		return err
	}

	if res.StatusCode() != 204 {
		msg := "Could not delete AppKey %s in consortium %s, in environment %s. Status: %d."
		return fmt.Errorf(msg, appKeyID, consortiumID, envID, res.StatusCode())
	}

	d.SetId("")
	return nil
}
