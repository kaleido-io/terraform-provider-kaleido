package kaleido

import (
	"fmt"

	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: resourceEnvironmentCreate,
		Read:   resourceEnvironmentRead,
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
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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
		},
	}
}

func resourceEnvironmentCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)

	if consortiumId == "" {
		return fmt.Errorf("Consortium missing id.")
	}
	environment := kaleido.NewEnvironment(d.Get("name").(string),
		d.Get("description").(string),
		d.Get("env_type").(string),
		d.Get("consensus_type").(string))
	res, err := client.CreateEnvironment(consortiumId, &environment)

	if err != nil {
		return err
	}

	if res.StatusCode() != 201 {
		msg := "Could not create environment %s for consortia %s, status was: %d"
		return fmt.Errorf(msg, environment.Name, consortiumId, res.StatusCode())
	}

	err = resource.Retry(d.Timeout("Create"), func() *resource.RetryError {
		res, retryErr := client.GetEnvironment(consortiumId, environment.Id, &environment)

		if retryErr != nil {
			return resource.NonRetryableError(retryErr)
		}

		statusCode := res.StatusCode()
		if statusCode != 200 {
			msg := fmt.Errorf("polling environment %s failed: %d", environment.Id, statusCode)
			return resource.NonRetryableError(msg)
		}

		if environment.State != "live" {
			msg := "Environment %s in consortium %s" +
				"took too long to enter state 'live'. Final state was '%s'."
			retryErr := fmt.Errorf(msg, environment.Id, consortiumId, environment.State)
			return resource.RetryableError(retryErr)
		}

		return nil
	})

	if err != nil {
		return err
	}

	d.SetId(environment.Id)
	return nil
}

func resourceEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	environmentId := d.Id()

	var environment kaleido.Environment
	res, err := client.GetEnvironment(consortiumId, environmentId, &environment)

	if err != nil {
		return err
	}

	if res.StatusCode() != 200 {
		msg := "Failed to get environment %s, from consortium %s status was: %d"
		return fmt.Errorf(msg, environmentId, consortiumId, res.StatusCode())
	}

	if res.StatusCode() == 404 {
		d.SetId("")
		return nil
	}

	d.Set("name", environment.Name)
	d.Set("description", environment.Description)
	d.Set("env_type", environment.Provider)
	d.Set("consensus_type", environment.ConsensusType)

	return nil
}

func resourceEnvironmentDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(kaleido.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	environmentId := d.Id()

	res, err := client.DeleteEnvironment(consortiumId, environmentId)

	if err != nil {
		return err
	}

	statusCode := res.StatusCode()
	if statusCode != 202 && statusCode != 204 {
		msg := "Failed to delete environment %s, in consortium %s, status was: %d"
		return fmt.Errorf(msg, environmentId, consortiumId, statusCode)
	}

	d.SetId("")

	return nil
}
