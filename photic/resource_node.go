package photic

import (
	"fmt"

	photic "github.com/Consensys/photic-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/schema"
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
		},
	}
}

func resourceNodeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(photic.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	environmentId := d.Get("environment_id").(string)
	membershipId := d.Get("membership_id").(string)
	node := photic.NewNode(d.Get("name").(string), membershipId)

	res, err := client.CreateNode(consortiumId, environmentId, &node)

	if err != nil {
		return err
	}

	status := res.StatusCode()
	if status != 201 {
		msg := "Could not create node %s in consortium %s in environment %s, status was: %d"
		return fmt.Errorf(msg, node.Name, consortiumId, environmentId, status)
	}

	d.SetId(node.Id)

	return nil
}

func resourceNodeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(photic.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	environmentId := d.Get("environment_id").(string)
	nodeId := d.Id()

	var node photic.Node
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
	return nil
}

func resourceNodeDelete(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("Not implemented.")
}
