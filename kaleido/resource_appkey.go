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

func resourceAppKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceAppKeyCreate,
		Read:   resourceAppKeyRead,
		Delete: resourceAppKeyDelete,
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

func resourceAppKeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	envId := d.Get("environment_id").(string)
	membershipId := d.Get("membership_id").(string)
	appKey := kaleido.NewAppKey(membershipId)

	res, err := client.CreateAppKey(consortiumId, envId, &appKey)

	if err != nil {
		return err
	}

	if res.StatusCode() != 201 {
		msg := "Could not create AppKey in consortium %s, in environment %s, with membership %s. Status: %d"
		return fmt.Errorf(msg, consortiumId, envId, membershipId, res.StatusCode())
	}

	d.SetId(appKey.Id)
	d.Set("username", appKey.Username)
	d.Set("password", appKey.Password)
	d.Set("auth_type", appKey.AuthType)

	return nil
}

func resourceAppKeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	envId := d.Get("environment_id").(string)
	appKeyId := d.Id()

	var appKey kaleido.AppKey
	res, err := client.GetAppKey(consortiumId, envId, appKeyId, &appKey)

	if err != nil {
		return err
	}

	if res.StatusCode() != 200 {
		msg := "Could not fetch AppKey %s in consortium %s, in environment %s. Status: %d."
		return fmt.Errorf(msg, appKeyId, consortiumId, envId, res.StatusCode())
	}

	d.Set("auth_type", appKey.AuthType)
	return nil
}

func resourceAppKeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	envId := d.Get("environment_id").(string)
	appKeyId := d.Id()

	res, err := client.DeleteAppKey(consortiumId, envId, appKeyId)

	if err != nil {
		return err
	}

	if res.StatusCode() != 204 {
		msg := "Could not fetch AppKey %s in consortium %s, in environment %s. Status: %d."
		return fmt.Errorf(msg, appKeyId, consortiumId, envId, res.StatusCode())
	}

	d.SetId("")
	return nil
}
