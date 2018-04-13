package photic

import (
	"fmt"

	photic "github.com/Consensys/photic-sdk-go/kaleido"
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
	client := meta.(photic.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)

	if consortiumId == "" {
		return fmt.Errorf("Consortium missing id.")
	}
	environment := photic.NewEnvironment(d.Get("name").(string),
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

	d.SetId(environment.Id)
	return nil
}

func resourceEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(photic.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	environmentId := d.Id()

	var environment photic.Environment
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
	client := meta.(photic.KaleidoClient)
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
