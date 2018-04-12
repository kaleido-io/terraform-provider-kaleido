package photic

import (
	"fmt"

	photic "github.com/Consensys/photic-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceConsortium() *schema.Resource {
	return &schema.Resource{
		Create: resourceConsortiumCreate,
		Read:   resourceConsortiumRead,
		Delete: resourceConsortiumDelete,
		Schema: map[string]*schema.Schema{
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
			"mode": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceConsortiumCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(photic.KaleidoClient)
	consortium := photic.NewConsortium(d.Get("name").(string),
		d.Get("description").(string),
		d.Get("mode").(string))
	res, err := client.CreateConsortium(&consortium)
	if err != nil {
		return err
	}
	status := res.StatusCode()
	if status != 201 {
		return fmt.Errorf("Failed to create consortium with status: %d", status)
	}

	d.SetId(consortium.Id)

	return nil
}

func resourceConsortiumRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(photic.KaleidoClient)
	var consortium photic.Consortium
	res, err := client.GetConsortium(d.Id(), &consortium)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 200 {
		if status == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Failed to read consortium with id %s status was: %d", d.Id(), status)
	}
	return nil
}

func resourceConsortiumDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(photic.KaleidoClient)
	res, err := client.DeleteConsortium(d.Id())

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status < 300 {
		fmt.Errorf("Failed to delete consortium with id %s status was %d.", d.Id(), status)
	}
	return nil
}
