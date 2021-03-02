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

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceNode() *schema.Resource {
	return &schema.Resource{
		Create: resourceNodeCreate,
		Read:   resourceNodeRead,
		Update: resourceNodeUpdate,
		Delete: resourceNodeDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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
			"role": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "validator",
			},
			"websocket_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"https_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"first_user_account": &schema.Schema{
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
			},
			"kms_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"opsmetric_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"backup_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"networking_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"node_config_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"baf_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func waitUntilNodeStarted(op, consortiumID, environmentID, nodeID string, node *kaleido.Node, d *schema.ResourceData, client kaleido.KaleidoClient) error {
	return resource.Retry(d.Timeout(op), func() *resource.RetryError {
		res, retryErr := client.GetNode(consortiumID, environmentID, nodeID, node)

		if retryErr != nil {
			return resource.NonRetryableError(retryErr)
		}

		statusCode := res.StatusCode()
		if statusCode != 200 {
			msg := fmt.Errorf("Fetching node %s state failed: %d", node.ID, statusCode)
			return resource.NonRetryableError(msg)
		}

		if node.State != "started" {
			msg := "Node %s in environment %s in consortium %s" +
				"took too long to enter state 'started'. Final state was '%s'."
			retryErr := fmt.Errorf(msg, node.ID, environmentID, consortiumID, node.State)
			return resource.RetryableError(retryErr)
		}

		return nil
	})
}

func resourceNodeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	membershipID := d.Get("membership_id").(string)
	ezoneID := d.Get("zone_id").(string)

	node := kaleido.NewNode(d.Get("name").(string), membershipID, ezoneID)

	node.Size = d.Get("size").(string)
	node.KmsID = d.Get("kms_id").(string)
	node.OpsmetricID = d.Get("opsmetric_id").(string)
	node.BackupID = d.Get("backup_id").(string)
	node.NetworkingID = d.Get("networking_id").(string)
	node.NodeConfigID = d.Get("node_config_id").(string)
	node.BafID = d.Get("baf_id").(string)
	node.Role = d.Get("role").(string)

	res, err := client.CreateNode(consortiumID, environmentID, &node)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Could not create node %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, node.Name, consortiumID, environmentID, status, res.String())
	}

	err = waitUntilNodeStarted("Create", consortiumID, environmentID, node.ID, &node, d, client)
	if err != nil {
		return err
	}

	d.SetId(node.ID)
	d.Set("websocket_url", node.Urls.WSS)
	d.Set("https_url", node.Urls.RPC)
	d.Set("first_user_account", node.FirstUserAccount)

	return nil
}

func resourceNodeUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)

	node := kaleido.NewNode(d.Get("name").(string), "", "")

	node.Size = d.Get("size").(string)
	node.KmsID = d.Get("kms_id").(string)
	node.OpsmetricID = d.Get("opsmetric_id").(string)
	node.BackupID = d.Get("backup_id").(string)
	node.NetworkingID = d.Get("networking_id").(string)
	node.NodeConfigID = d.Get("node_config_id").(string)
	node.BafID = d.Get("baf_id").(string)
	nodeID := d.Id()

	res, err := client.UpdateNode(consortiumID, environmentID, nodeID, &node)
	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		msg := "Could not update node %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, nodeID, consortiumID, environmentID, status, res.String())
	}

	res, err = client.ResetNode(consortiumID, environmentID, node.ID)
	if err != nil {
		return err
	}
	if status != 200 {
		msg := "Could not reset node %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, nodeID, consortiumID, environmentID, status, res.String())
	}

	err = waitUntilNodeStarted("Update", consortiumID, environmentID, node.ID, &node, d, client)
	if err != nil {
		return err
	}

	return nil
}

func resourceNodeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	nodeID := d.Id()

	var node kaleido.Node
	res, err := client.GetNode(consortiumID, environmentID, nodeID, &node)

	if err != nil {
		return err
	}

	status := res.StatusCode()

	if status == 404 {
		d.SetId("")
		return nil
	}
	if status != 200 {
		msg := "Could not find node %s in consortium %s in environment %s with status %d: %s"
		return fmt.Errorf(msg, nodeID, consortiumID, environmentID, status, res.String())
	}

	d.Set("name", node.Name)
	d.Set("role", node.Role)
	d.Set("websocket_url", node.Urls.WSS)
	d.Set("https_url", node.Urls.RPC)
	d.Set("first_user_account", node.FirstUserAccount)
	return nil
}

func resourceNodeDelete(d *schema.ResourceData, meta interface{}) error {

	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Get("environment_id").(string)
	nodeID := d.Id()

	res, err := client.DeleteNode(consortiumID, environmentID, nodeID)

	if err != nil {
		return err
	}

	statusCode := res.StatusCode()
	if res.IsError() && statusCode != 404 {
		msg := "Failed to delete node %s in environment %s in consortium %s with status %d: %s"
		return fmt.Errorf(msg, nodeID, environmentID, consortiumID, statusCode, res.String())
	}

	d.SetId("")

	return nil
}
