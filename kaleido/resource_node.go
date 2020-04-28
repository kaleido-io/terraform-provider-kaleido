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

func resourceNode() *schema.Resource {
	return &schema.Resource{
		Create: resourceNodeCreate,
		Read:   resourceNodeRead,
		Delete: resourceNodeDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
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
			"websocker_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"https_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"size": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceNodeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	environmentId := d.Get("environment_id").(string)
	membershipId := d.Get("membership_id").(string)
	ezoneId := d.Get("zone_id").(string)
	size := d.Get("size").(string)
	node := kaleido.NewNode(d.Get("name").(string), membershipId, ezoneId)
	node.Size = size

	res, err := client.CreateNode(consortiumId, environmentId, &node)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Could not create node %s in consortium %s in environment %s, status was: %d"
		return fmt.Errorf(msg, node.Id, consortiumId, environmentId, status)
	}

	err = resource.Retry(d.Timeout("Create"), func() *resource.RetryError {
		res, retryErr := client.GetNode(consortiumId, environmentId, node.Id, &node)

		if retryErr != nil {
			return resource.NonRetryableError(retryErr)
		}

		statusCode := res.StatusCode()
		if statusCode != 200 {
			msg := fmt.Errorf("Fetching node %s state failed: %d", node.Id, statusCode)
			return resource.NonRetryableError(msg)
		}

		if node.State != "started" {
			msg := "Node %s in environment %s in consortium %s" +
				"took too long to enter state 'started'. Final state was '%s'."
			retryErr := fmt.Errorf(msg, node.Id, environmentId, consortiumId, node.State)
			return resource.RetryableError(retryErr)
		}

		return nil
	})

	if err != nil {
		return err
	}

	d.SetId(node.Id)
	d.Set("websocket_url", node.Urls.WSS)
	d.Set("https_url", node.Urls.RPC)

	return nil
}

func resourceNodeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	environmentId := d.Get("environment_id").(string)
	nodeId := d.Id()

	var node kaleido.Node
	res, err := client.GetNode(consortiumId, environmentId, nodeId, &node)

	if err != nil {
		return err
	}

	status := res.StatusCode()

	if status == 404 {
		d.SetId("")
		return nil
	}
	if status != 200 {
		msg := "Could not find node %s in consortium %s in environment %s, status: %d"
		return fmt.Errorf(msg, nodeId, consortiumId, environmentId, status)
	}

	d.Set("name", node.Name)
	d.Set("websocket_url", node.Urls.WSS)
	d.Set("https_url", node.Urls.RPC)
	return nil
}

func resourceNodeDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
