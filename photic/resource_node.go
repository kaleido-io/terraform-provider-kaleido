package photic

import (
	"fmt"
	"time"

	photic "github.com/Consensys/photic-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/resource"
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
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
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

	err = resource.Retry(d.Timeout("Create"), func() *resource.RetryError {
		var nodeState photic.Node
		res, err := client.GetNode(consortiumId, environmentId, node.Id, &nodeState)

		if err != nil {
			return resource.NonRetryableError(err)
		}

		statusCode := res.StatusCode()
		if statusCode != 200 {
			msg := fmt.Errorf("Fetching node state failed: %d", statusCode)
			return resource.NonRetryableError(msg)
		}

		if nodeState.State != "started" {
			msg := "Node %s in environment %s in consortium %s" +
				"took too long to enter state 'started'. Final state was %s."
			err := fmt.Errorf(msg, nodeState.State, environmentId, consortiumId, nodeState.State)
			return resource.RetryableError(err)
		}

		return nil
	})

	if err != nil {
		return err
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
	d.SetId("")
	return nil
}
