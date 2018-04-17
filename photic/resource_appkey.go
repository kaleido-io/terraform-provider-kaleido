package photic

import (
	"fmt"

	photic "github.com/Consensys/photic-sdk-go/kaleido"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAppKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceAppKeyCreate,
		Read:   resourceAppKeyRead,
		Delete: resourceAppKeyDelete,
		Schema: map[string]*schema.Schema{
			"membership_id": &schema.Schema{
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
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"password": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"auth_type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAppKeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(photic.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	envId := d.Get("environment_id").(string)
	membershipId := d.Get("membership_id").(string)
	appKey := photic.NewAppKey(membershipId)

	res, err := client.CreateAppKey(consortiumId, envId, &appKey)

	if err != nil {
		return err
	}

	if res.StatusCode() != 201 {
		msg := "Could not create AppKey in consortium %s, in environment %s, with membership %s. Status: %d"
		return fmt.Errorf(msg, consortiumId, envId, membershipId, res.StatusCode())
	}

	d.SetId(appKey.Id)
	d.Set("username", appKey.Username)
	d.Set("password", appKey.Password)
	d.Set("auth_type", appKey.AuthType)

	return nil
}

func resourceAppKeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(photic.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	envId := d.Get("environment_id").(string)
	appKeyId := d.Id()

	var appKey photic.AppKey
	res, err := client.GetAppKey(consortiumId, envId, appKeyId, &appKey)

	if err != nil {
		return err
	}

	if res.StatusCode() != 200 {
		msg := "Could not fetch AppKey %s in consortium %s, in environment %s. Status: %d."
		return fmt.Errorf(msg, appKeyId, consortiumId, envId, res.StatusCode())
	}

	d.Set("username", appKey.Username)
	d.Set("auth_type", appKey.AuthType)
	return nil
}

func resourceAppKeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(photic.KaleidoClient)
	consortiumId := d.Get("consortium_id").(string)
	envId := d.Get("environment_id").(string)
	appKeyId := d.Id()

	res, err := client.DeleteAppKey(consortiumId, envId, appKeyId)

	if err != nil {
		return err
	}

	if res.StatusCode() != 204 {
		msg := "Could not fetch AppKey %s in consortium %s, in environment %s. Status: %d."
		return fmt.Errorf(msg, appKeyId, consortiumId, envId, res.StatusCode())
	}

	d.SetId("")
	return nil
}
