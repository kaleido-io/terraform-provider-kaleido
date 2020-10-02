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

func resourceService() *schema.Resource {
	return &schema.Resource{
		Create: resourceServiceCreate,
		Read:   resourceServiceRead,
		Delete: resourceServiceDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_type": &schema.Schema{
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
			"membership_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"details": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"https_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"websocket_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"webui_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceServiceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	membershipID := d.Get("membership_id").(string)
	serviceType := d.Get("service_type").(string)
	details := d.Get("details").(map[string]interface{})
	zoneID := d.Get("zone_id").(string)
	service := kaleido.NewService(d.Get("name").(string), serviceType, membershipID, zoneID, details)

	res, err := client.CreateService(consortiumID, environmentID, &service)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Could not create service %s in consortium %s in environment %s, status was: %d, error: %s"
		return fmt.Errorf(msg, service.ID, consortiumID, environmentID, status, res.String())
	}

	err = resource.Retry(d.Timeout("Create"), func() *resource.RetryError {
		res, retryErr := client.GetService(consortiumID, environmentID, service.ID, &service)

		if retryErr != nil {
			return resource.NonRetryableError(retryErr)
		}

		statusCode := res.StatusCode()
		if statusCode != 200 {
			msg := fmt.Errorf("Fetching service %s state failed: %d", service.ID, statusCode)
			return resource.NonRetryableError(msg)
		}

		if service.State != "started" {
			msg := "Service %s in environment %s in consortium %s" +
				"took too long to enter state 'started'. Final state was '%s'."
			retryErr := fmt.Errorf(msg, service.ID, environmentID, consortiumID, service.State)
			return resource.RetryableError(retryErr)
		}

		return nil
	})

	if err != nil {
		return err
	}

	d.SetId(service.ID)
	d.Set("https_url", service.Urls["http"])
	if wsURL, ok := service.Urls["ws"]; ok {
		d.Set("websocket_url", wsURL)
	}
	if webuiURL, ok := service.Urls["webui"]; ok {
		d.Set("webui_url", webuiURL)
	}
	return nil
}

func resourceServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	serviceID := d.Id()

	var service kaleido.Service
	res, err := client.GetService(consortiumID, environmentID, serviceID, &service)

	if err != nil {
		return err
	}

	status := res.StatusCode()

	if status == 404 {
		d.SetId("")
		return nil
	}
	if status != 200 {
		msg := "Could not find service %s in consortium %s in environment %s, status: %d"
		return fmt.Errorf(msg, serviceID, consortiumID, environmentID, status)
	}

	d.Set("name", service.Name)
	d.Set("service_type", service.Service)
	d.Set("https_url", service.Urls["http"])
	if wsURL, ok := service.Urls["ws"]; ok {
		d.Set("websocket_url", wsURL)
	}
	if webuiURL, ok := service.Urls["webui"]; ok {
		d.Set("webui_url", webuiURL)
	}
	return nil
}

func resourceServiceDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
