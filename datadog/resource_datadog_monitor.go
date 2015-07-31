package datadog

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/zorkian/go-datadog-api"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDatadogMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogMonitorCreate,
		Read:   resourceDatadogMonitorRead,
		Update: resourceDatadogMonitorUpdate,
		Delete: resourceDatadogMonitorDelete,
		Exists: resourceDatadogMonitorExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			// Metric and Monitor settings
			"metric": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"metric_tags": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "*",
			},
			"time_aggr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"time_window": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"space_aggr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"operator": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"message": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			// Alert Settings
			"warning": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
			},
			"critical": &schema.Schema{
				Type:     schema.TypeMap,
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
		},
	}
}

// TODO: Rename this one?
func buildMonitorStruct(d *schema.ResourceData, typeStr string) *datadog.Monitor {
	name := d.Get("name").(string)
	message := d.Get("message").(string)
	timeAggr := d.Get("time_aggr").(string)
	timeWindow := d.Get("time_window").(string)
	spaceAggr := d.Get("space_aggr").(string)
	metric := d.Get("metric").(string)
	tags := d.Get("metric_tags").(string)
	operator := d.Get("operator").(string)
	query := fmt.Sprintf("%s(%s):%s:%s{%s} %s %s", timeAggr, timeWindow, spaceAggr, metric, tags, operator, d.Get(fmt.Sprintf("%s.threshold", typeStr)))

	log.Println(query)

	o := datadog.Options{
		NotifyNoData: d.Get("notify_no_data").(bool),
		NoDataTimeframe: d.Get("no_data_timeframe").(int),
	}

	m := datadog.Monitor{
		Type:    "metric alert",
		Query:   query,
		Name:    fmt.Sprintf("[%s] %s", typeStr, name),
		Message: fmt.Sprintf("%s %s", message, d.Get(fmt.Sprintf("%s.notify", typeStr))),
		Options: o,
	}

	return &m
}

func resourceDatadogMonitorCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)
	log.Printf("[DEBUG] running create.")

	w, w_err := client.CreateMonitor(buildMonitorStruct(d, "warning"))

	if w_err != nil {
		return fmt.Errorf("error creating warning: %s", w_err)
	}

	c, c_err := client.CreateMonitor(buildMonitorStruct(d, "critical"))

	if c_err != nil {
		return fmt.Errorf("error creating warning: %s", c_err)
	}

	log.Printf("[DEBUG] Saving IDs: %s__%s", strconv.Itoa(w.Id), strconv.Itoa(c.Id))

	d.SetId(fmt.Sprintf("%s__%s", strconv.Itoa(w.Id), strconv.Itoa(c.Id)))

	return nil
}

func resourceDatadogMonitorDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	log.Printf("[DEBUG] running delete.")

	for _, v := range strings.Split(d.Id(), "__") {
		if v == "" {
			return fmt.Errorf("Id not set.")
		}
		Id, i_err := strconv.Atoi(v)

		if i_err != nil {
			return i_err
		}

		err := client.DeleteMonitor(Id)
		if err != nil {
			return err
		}
	}
	return nil
}

func resourceDatadogMonitorExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	// Exists - This is called to verify a resource still exists. It is called prior to Read,
	// and lowers the burden of Read to be able to assume the resource exists.

	client := meta.(*datadog.Client)

	log.Printf("[DEBUG] running exists.")

	// Sanitise this one
	exists := false
	for _, v := range strings.Split(d.Id(), "__") {
		if v == "" {
			log.Printf("[DEBUG] Could not parse IDs. %s", v)
			return false, fmt.Errorf("Id not set.")
		}
		Id, i_err := strconv.Atoi(v)

		if i_err != nil {
			log.Printf("[DEBUG] Received error converting string %s", i_err)
			return false, i_err
		}
		_, err := client.GetMonitor(Id)
		if err != nil {
			// Monitor did does not exist, continue.
			log.Printf("[DEBUG] monitor does not exist. %s", err)
			e = err
			continue
		}
		exists = true
	}

	if exists == false {
		return false, nil
	}

	return true, nil
}

func resourceDatadogMonitorRead(d *schema.ResourceData, meta interface{}) error {
	// TODO: add support for this a read function.
	/* Read - This is called to resync the local state with the remote state.
	 Terraform guarantees that an existing ID will be set. This ID should be
	 used to look up the resource. Any remote data should be updated into the
	 local data. No changes to the remote resource are to be made.
	*/

	return nil
}

func resourceDatadogMonitorUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] running update.")

	split := strings.Split(d.Id(), "__")

	wID, cID := split[0], split[1]

	if wID == "" {
		return fmt.Errorf("Id not set.")
	}

	if cID == "" {
		return fmt.Errorf("Id not set.")
	}

	warningId, i_err := strconv.Atoi(wID)

	if i_err != nil {
		return i_err
	}

	criticalId, i_err := strconv.Atoi(cID)

	if i_err != nil {
		return i_err
	}


	client := meta.(*datadog.Client)

	warning_body := buildMonitorStruct(d, "warning")
	critical_body := buildMonitorStruct(d, "critical")

	warning_body.Id = warningId
	critical_body.Id = criticalId

	w_err := client.UpdateMonitor(warning_body)

	if w_err != nil {
		return fmt.Errorf("error updating warning: %s", w_err.Error())
	}

	c_err := client.UpdateMonitor(critical_body)

	if c_err != nil {
		return fmt.Errorf("error updating critical: %s", c_err.Error())
	}

	return nil
}
