package datadog

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

// resourceDatadogServiceCheck is a Datadog monitor resource
func resourceDatadogServiceCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogServiceCheckCreate,
		Read:   resourceDatadogServiceCheckRead,
		Update: resourceDatadogServiceCheckUpdate,
		Delete: resourceDatadogServiceCheckDelete,
		Exists: resourceDatadogServiceCheckExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"check": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"check_count": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			// Metric and ServiceCheck settings
			"metric": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"tags": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "*",
			},
			"message": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			// Additional Settings
			"notify_no_data": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"no_data_timeframe": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"renotify_interval": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
		},
	}
}

// buildServiceCheckStruct returns a monitor struct
func buildServiceCheckStruct(d *schema.ResourceData) *datadog.Monitor {
	log.Print("[DEBUG] building monitor struct")
	name := d.Get("name").(string)
	message := d.Get("message").(string)
	tags := d.Get("tags").(string)
	var monitorName string
	var query string

	check := d.Get("check").(string)
	checkCount := d.Get("check_count").(string)
	query = fmt.Sprintf("\"%s\".over(\"%s\").last(%s).count_by_status()", check, tags, checkCount)
	monitorName = name

	o := datadog.Options{
		NotifyNoData:     d.Get("notify_no_data").(bool),
		NoDataTimeframe:  d.Get("no_data_timeframe").(int),
		RenotifyInterval: d.Get("renotify_interval").(int),
	}

	m := datadog.Monitor{
		Type:    "service check",
		Query:   query,
		Name:    monitorName,
		Message: fmt.Sprintf("%s", message),
		Options: o,
	}

	return &m
}

// resourceDatadogServiceCheckCreate creates a monitor.
func resourceDatadogServiceCheckCreate(d *schema.ResourceData, meta interface{}) error {
	log.Print("[DEBUG] creating monitor")
	client := meta.(*datadog.Client)

	log.Print("[DEBUG] Creating service check")
	m, err := client.CreateMonitor(buildServiceCheckStruct(d))

	if err != nil {
		return fmt.Errorf("error creating service check: %s", err)
	}

	d.SetId(strconv.Itoa(m.Id))
	return nil
}

// resourceDatadogServiceCheckDelete deletes a monitor.
func resourceDatadogServiceCheckDelete(d *schema.ResourceData, meta interface{}) error {
	log.Print("[DEBUG] deleting monitor")
	client := meta.(*datadog.Client)

	log.Print("[DEBUG] Deleting service check")
	ID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	err = client.DeleteMonitor(ID)
	if err != nil {
		return err
	}
	return nil
}

// resourceDatadogServiceCheckExists verifies a monitor exists.
func resourceDatadogServiceCheckExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	// Exists - This is called to verify a resource still exists. It is called prior to Read,
	// and lowers the burden of Read to be able to assume the resource exists.

	log.Print("[DEBUG] verifying monitor exists")
	client := meta.(*datadog.Client)

	log.Print("[DEBUG] verifying service check exists")
	ID, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, err
	}
	_, err = client.GetMonitor(ID)

	if err != nil {
		if strings.EqualFold(err.Error(), "API error: 404 Not Found") {
			log.Printf("[DEBUG] monitor does not exist: %s", err)
			return false, err
		}
		return false, err
	}

	return true, nil
}

// resourceDatadogServiceCheckRead synchronises Datadog and local state .
func resourceDatadogServiceCheckRead(d *schema.ResourceData, meta interface{}) error {
	// TODO: add support for this a read function.
	/* Read - This is called to resync the local state with the remote state.
	Terraform guarantees that an existing ID will be set. This ID should be
	used to look up the resource. Any remote data should be updated into the
	local data. No changes to the remote resource are to be made.
	*/

	return nil
}

// resourceDatadogServiceCheckUpdate updates a monitor.
func resourceDatadogServiceCheckUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] running update.")

	client := meta.(*datadog.Client)

	body := buildServiceCheckStruct(d)

	ID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	body.Id = ID
	err = client.UpdateMonitor(body)

	if err != nil {
		return fmt.Errorf("error updating warning: %s", err.Error())
	}
	return nil
}
