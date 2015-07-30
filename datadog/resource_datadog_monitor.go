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

	w, w_err := client.CreateMonitor(buildMonitorStruct(d, "warning"))

	if w_err != nil {
		return fmt.Errorf("error creating warning: %s", w_err)
	}

	c, c_err := client.CreateMonitor(buildMonitorStruct(d, "critical"))

	if c_err != nil {
		return fmt.Errorf("error creating warning: %s", c_err)
	}

	d.SetId(fmt.Sprintf("%s__%s", w, c))

	return nil
}

func resourceDatadogMonitorDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

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
	client := meta.(*datadog.Client)

	b = true
	for _, v := range strings.Split(d.Id(), "__") {
		if v == "" {
			return false, fmt.Errorf("Id not set.")
		}
		Id, i_err := strconv.Atoi(v)

		if i_err != nil {
			return false, i_err
		}
		_, err := client.GetMonitor(Id)
		if err != nil {
			// There is an error, we go on to the next
			e = err
			continue
		}
		b = b && true
	}
	if !b {
		return false, resourceDatadogMonitorDelete(d, meta)
	}

	return false, nil
}

func resourceDatadogMonitorRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceDatadogMonitorUpdate(d *schema.ResourceData, meta interface{}) error {
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
