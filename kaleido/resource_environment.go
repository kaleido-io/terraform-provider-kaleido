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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

func resourceEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: resourceEnvironmentCreate,
		Read:   resourceEnvironmentRead,
		Update: resourceEnvironmentUpdate,
		Delete: resourceEnvironmentDelete,
		Schema: map[string]*schema.Schema{
			"consortium_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: false,
				Default:  "",
				Optional: true,
			},
			"shared_deployment": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "The decentralized nature of Kaleido means an environment might be shared with other accounts. When true only create if name does not exist, and delete becomes a no-op.",
			},
			"env_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"consensus_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"release_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
			"multi_region": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"block_period": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"prefunded_accounts": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"test_features_json": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func setTestFeatures(d *schema.ResourceData, client kaleido.KaleidoClient, environment *kaleido.Environment) error {
	testFeaturesJSON := d.Get("test_features_json").(string)
	if testFeaturesJSON != "" {
		var testFeatures map[string]interface{}
		if err := json.Unmarshal([]byte(testFeaturesJSON), &testFeatures); err != nil {
			return fmt.Errorf("Failed to parse test_features_json: %s", err)
		}
		for k, v := range testFeatures {
			environment.TestFeatures[k] = v
		}
	}
	return nil
}

func resourceEnvironmentCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	prefundedAccounts := d.Get("prefunded_accounts").(map[string]interface{})
	prefundedAccountsStringified := map[string]string{}
	for key, val := range prefundedAccounts {
		valStr, ok := val.(string)
		if !ok {
			return fmt.Errorf("Unable to read balance of pre-funded account: %s", key)
		}
		prefundedAccountsStringified[key] = valStr
	}

	if consortiumID == "" {
		return fmt.Errorf("Consortium missing id.")
	}
	environment := kaleido.NewEnvironment(d.Get("name").(string),
		d.Get("description").(string),
		d.Get("env_type").(string),
		d.Get("consensus_type").(string),
		d.Get("multi_region").(bool),
		d.Get("block_period").(int),
		prefundedAccountsStringified)

	if err := setTestFeatures(d, client, &environment); err != nil {
		return err
	}

	releaseID, ok := d.GetOk("release_id")

	if ok {
		environment.ReleaseID = releaseID.(string)
	}

	if d.Get("shared_deployment").(bool) {
		var environments []kaleido.Environment
		res, err := client.ListEnvironments(consortiumID, &environments)
		if err != nil {
			return err
		}
		if res.StatusCode() != 200 {
			return fmt.Errorf("Failed to list existing environments with status %d: %s", res.StatusCode(), res.String())
		}
		for _, e := range environments {
			if e.Name == environment.Name && !strings.Contains(e.State, "delete") {
				// Already exists, just re-use
				d.SetId(e.ID)
				return resourceEnvironmentRead(d, meta)
			}
		}
	}

	res, err := client.CreateEnvironment(consortiumID, &environment)

	if err != nil {
		return err
	}

	if res.StatusCode() != 201 {
		msg := "Could not create environment %s for consortia %s with status %d: %s"
		return fmt.Errorf(msg, environment.Name, consortiumID, res.StatusCode(), res.String())
	}

	err = resource.Retry(d.Timeout("Create"), func() *resource.RetryError {
		res, retryErr := client.GetEnvironment(consortiumID, environment.ID, &environment)

		if retryErr != nil {
			return resource.NonRetryableError(retryErr)
		}

		statusCode := res.StatusCode()
		if statusCode != 200 {
			msg := fmt.Errorf("polling environment %s failed: %d", environment.ID, statusCode)
			return resource.NonRetryableError(msg)
		}

		return nil
	})

	if err != nil {
		return err
	}

	d.SetId(environment.ID)
	if environment.ReleaseID != "" {
		d.Set("release_id", environment.ReleaseID)
	}
	return nil
}

func resourceEnvironmentUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Id()

	environment := kaleido.NewEnvironment(
		d.Get("name").(string),
		d.Get("description").(string),
		"",    // cannot change
		"",    // cannot change
		false, // cannot change
		0,     // cannot change
		nil)

	if err := setTestFeatures(d, client, &environment); err != nil {
		return err
	}

	res, err := client.UpdateEnvironment(consortiumID, environmentID, &environment)

	if err != nil {
		return err
	}

	statusCode := res.StatusCode()
	if statusCode != 200 {
		msg := "Failed to update environment %s, in consortium %s with status %d: %s"
		return fmt.Errorf(msg, environmentID, consortiumID, statusCode, res.String())
	}

	return nil
}

func resourceEnvironmentDelete(d *schema.ResourceData, meta interface{}) error {
	if d.Get("shared_deployment").(bool) {
		// Cannot safely delete if this is shared with other terraform deployments
		d.SetId("")
		return nil
	}

	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Id()

	res, err := client.DeleteEnvironment(consortiumID, environmentID)

	if err != nil {
		return err
	}

	statusCode := res.StatusCode()
	if statusCode != 202 && statusCode != 204 {
		msg := "Failed to delete environment %s, in consortium %s with status %d: %s"
		return fmt.Errorf(msg, environmentID, consortiumID, statusCode, res.String())
	}

	d.SetId("")

	return nil
}

func resourceEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumID := d.Get("consortium_id").(string)
	environmentID := d.Id()

	var environment kaleido.Environment
	res, err := client.GetEnvironment(consortiumID, environmentID, &environment)

	if err != nil {
		return err
	}

	if res.StatusCode() != 200 {
		msg := "Failed to get environment %s, from consortium %s with status %d: %s"
		return fmt.Errorf(msg, environmentID, consortiumID, res.StatusCode(), res.String())
	}

	if res.StatusCode() == 404 {
		d.SetId("")
		return nil
	}

	d.Set("name", environment.Name)
	d.Set("description", environment.Description)
	d.Set("env_type", environment.Provider)
	d.Set("consensus_type", environment.ConsensusType)
	d.Set("release_id", environment.ReleaseID)
	balances := map[string]string{}
	for account, balanceDetails := range environment.PrefundedAccounts {
		values := balanceDetails.(map[string]interface{})
		balanceStr := ""
		for _, balance := range values {
			balanceStr = balance.(string)
		}
		balances[account] = balanceStr
	}
	d.Set("prefunded_accounts", balances)

	return nil
}
