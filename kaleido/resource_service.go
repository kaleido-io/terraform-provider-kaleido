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
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceService() *schema.Resource {
	return &schema.Resource{
		Create: resourceServiceCreate,
		Read:   resourceServiceRead,
		Update: resourceServiceUpdate,
		Delete: resourceServiceDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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
			"shared_deployment": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "The decentralized nature of Kaleido means a utility service might be shared with other accounts. When true only create if service_type does not exist, and delete becomes a no-op.",
			},
			"size": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"details": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
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
			"urls": &schema.Schema{
				Type:     schema.TypeMap,
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

func waitUntilServiceStarted(op, consortiumID, environmentID, serviceID string, service *kaleido.Service, d *schema.ResourceData, client kaleido.KaleidoClient) error {
	return resource.Retry(d.Timeout(op), func() *resource.RetryError {
		res, retryErr := client.GetService(consortiumID, environmentID, service.ID, service)

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
}

func setServiceUrls(d *schema.ResourceData, service *kaleido.Service) {
	urls := make(map[string]string)
	for name, urlValue := range service.Urls {
		if urlString, ok := urlValue.(string); ok {
			urls[name] = urlString
		}
	}
	d.Set("urls", urls)
}

func resourceServiceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	membershipID := d.Get("membership_id").(string)
	serviceType := d.Get("service_type").(string)
	details := duplicateDetails(d.Get("details").(map[string]interface{}))
	zoneID := d.Get("zone_id").(string)
	service := kaleido.NewService(d.Get("name").(string), serviceType, membershipID, zoneID, details)
	service.Size = d.Get("size").(string)

	if d.Get("shared_deployment").(bool) {
		var existing []kaleido.Service
		res, err := client.ListServices(consortiumID, environmentID, &existing)
		if err != nil {
			return err
		}
		if res.StatusCode() != 200 {
			return fmt.Errorf("Failed to list existing services with status %d: %s", res.StatusCode(), res.String())
		}
		for _, e := range existing {
			if e.Service == service.Service && !strings.Contains(e.State, "delete") {
				if e.ServiceType != "utility" {
					return fmt.Errorf("The shared_deployment option only applies to utility services. %s service %s is a '%s' service", service.Service, service.ID, service.ServiceType)
				}
				// Already exists, just re-use
				d.SetId(e.ID)
				return resourceServiceRead(d, meta)
			}
		}
	}

	res, err := client.CreateService(consortiumID, environmentID, &service)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Could not create service %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, service.ID, consortiumID, environmentID, status, res.String())
	}

	err = waitUntilServiceStarted("Create", consortiumID, environmentID, service.ID, &service, d, client)

	if err != nil {
		return err
	}

	d.SetId(service.ID)
	setServiceUrls(d, &service)
	d.Set("https_url", service.Urls["http"])
	if wsURL, ok := service.Urls["ws"]; ok {
		d.Set("websocket_url", wsURL)
	}
	if webuiURL, ok := service.Urls["webui"]; ok {
		d.Set("webui_url", webuiURL)
	}
	return nil
}

func duplicateDetails(detailsSubmitted map[string]interface{}) map[string]interface{} {
	// We do not want to save back updates that come back over the rest API into the terraform
	// state, otherwise we will think there is a difference between any generated sub-fields
	// inside of the details structure, and the next terraform apply will attempt to perform an update.
	details := make(map[string]interface{})
	for k, v := range detailsSubmitted {
		details[k] = v
	}
	return details
}

func resourceServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	details := duplicateDetails(d.Get("details").(map[string]interface{}))
	service := kaleido.NewService(d.Get("name").(string), "", "", "", details)
	service.Size = d.Get("size").(string)
	serviceID := d.Id()

	res, err := client.UpdateService(consortiumID, environmentID, serviceID, &service)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Could not update service %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, serviceID, consortiumID, environmentID, status, res.String())
	}

	res, err = client.ResetService(consortiumID, environmentID, service.ID)
	if err != nil {
		return err
	}
	if status != 200 {
		msg := "Could not reset service %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, serviceID, consortiumID, environmentID, status, res.String())
	}

	err = waitUntilServiceStarted("Update", consortiumID, environmentID, serviceID, &service, d, client)

	if err != nil {
		return err
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
		msg := "Could not find service %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, serviceID, consortiumID, environmentID, status, res.String())
	}

	d.Set("name", service.Name)
	d.Set("service_type", service.Service)
	setServiceUrls(d, &service)
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
	if d.Get("shared_deployment").(bool) {
		// Cannot safely delete if this is shared with other terraform deployments
		d.SetId("")
		return nil
	}

	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	serviceID := d.Id()

	res, err := client.DeleteService(consortiumID, environmentID, serviceID)

	if err != nil {
		return err
	}

	statusCode := res.StatusCode()
	if res.IsError() && statusCode != 404 {
		msg := "Failed to delete service %s in environment %s in consortium %s with status %d: %s"
		return fmt.Errorf(msg, serviceID, environmentID, consortiumID, statusCode, res.String())
	}

	d.SetId("")

	return nil
}
