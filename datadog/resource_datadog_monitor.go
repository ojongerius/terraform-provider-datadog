package datadog

import (
	"fmt"
	"log"
	"strconv"

	"encoding/json"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

// resourceDatadogMonitor is a Datadog monitor resource
func resourceDatadogMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogMonitorCreate,
		Read:   resourceDatadogMonitorRead,
		Update: resourceDatadogMonitorUpdate,
		Delete: resourceDatadogGenericDelete,
		Exists: resourceDatadogGenericExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"message": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"query": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"thresholds": thresholdSchema(),

			// Options
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

			"notify_audit": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"period": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  false,
			},
			/*
				TODO: implement this
				Silenced          map[string]int `json:"silenced,omitempty"`
				"silenced": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
					Default:  false,
				},
			*/
			"timeout_h": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"escalation_message": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			/*
							TODO: implement these

								tags [optional, default=empty list]
								A list of tags to associate with your monitor. This can help you categorize and filter monitors.
								options [optional]
								A dictionary of options for the monitor. There are options that are common to all types as well as options that are specific to certain monitor types.
								COMMON OPTIONS

								silenced dictionary of scopes to timestamps or None. Each scope will be muted until the given POSIX timestamp or forever if the value is None.
								Default: None

								Examples:

								To mute the alert completely:
								{'*': None}

								To mute role:db for a short time:
								{'role:db': 1412798116}

				                TODO: needs upstream change
								include_tags a boolean indicating whether notifications from this monitor will automatically insert its triggering tags into the title.
								Default: True
			*/
		},
	}
}

// buildMonitorStruct returns a monitor struct
func buildMonitorStruct(d *schema.ResourceData) *datadog.Monitor {

	_, thresholds := getThresholds(d)

	// Not all of above options are mandatory, consider doing GetOk for all these,
	// unless they have defaults or are mandatory

	o := datadog.Options{
		NotifyNoData:      d.Get("notify_no_data").(bool),
		NoDataTimeframe:   d.Get("no_data_timeframe").(int),
		RenotifyInterval:  d.Get("renotify_interval").(int),
		Thresholds:        thresholds,
		NotifyAudit:       d.Get("notify_audit").(bool),
		Period:            d.Get("period").(int),
		TimeoutH:          d.Get("timeout_h").(int),
		EscalationMessage: d.Get("escalation_message").(string),
	}

	m := datadog.Monitor{
		Type:    d.Get("type").(string),
		Query:   d.Get("query").(string),
		Name:    d.Get("name").(string),
		Message: d.Get("message").(string),
		Options: o,
	}

	return &m
}

// resourceDatadogMonitorCreate creates a monitor.
func resourceDatadogMonitorCreate(d *schema.ResourceData, meta interface{}) error {

	m := buildMonitorStruct(d)
	if err := monitorCreator(d, meta, m); err != nil {
		return err
	}

	return nil
}

// resourceDatadogMonitorRead creates a monitor.
func resourceDatadogMonitorRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	// Workaround to handle upgrades from < 0.0.4

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		// TODO: dress/decorate error
		return err
	}

	m, err := client.GetMonitor(i)

	if err != nil {
		// TODO: dress/decorate error
		return err
	}

	log.Printf("[DEBUG] monitor: %v", m)
	d.Set("name", m.Name)
	d.Set("message", m.Message)
	d.Set("query", m.Query)
	d.Set("type", m.Type)
	d.Set("thresholds", m.Options.Thresholds)
	d.Set("notify_no_data", m.Options.NotifyNoData)
	d.Set("notify_no_data_timeframe", m.Options.NoDataTimeframe)
	d.Set("renotify_interval", m.Options.RenotifyInterval)
	d.Set("notify_audit", m.Options.NotifyAudit)
	d.Set("period", m.Options.Period)
	d.Set("timeout_h", m.Options.TimeoutH)
	d.Set("escalation_message", m.Options.EscalationMessage)

	return nil
}

// resourceDatadogMonitorUpdate updates a monitor.
func resourceDatadogMonitorUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Print("[DEBUG] running update.")
	client := meta.(*datadog.Client)

	m := &datadog.Monitor{}

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	m.Id = i

	if attr, ok := d.GetOk("name"); ok {
		m.Name = attr.(string)
	}
	if attr, ok := d.GetOk("message"); ok {
		m.Message = attr.(string)
	}
	if attr, ok := d.GetOk("query"); ok {
		m.Query = attr.(string)
	}

	o := datadog.Options{}

	if attr, ok := d.GetOk("thresholds"); ok {
		thresholds := attr.(map[string]interface{})
		if thresholds["ok"] != nil {
			o.Thresholds.Ok = json.Number(thresholds["ok"].(string))
		}
		if thresholds["warning"] != nil {
			o.Thresholds.Warning = json.Number(thresholds["warning"].(string))
		}
		if thresholds["critical"] != nil {
			o.Thresholds.Critical = json.Number(thresholds["critical"].(string))
		}
	}

	if attr, ok := d.GetOk("notify_no_data"); ok {
		o.NotifyNoData = attr.(bool)
	}
	if attr, ok := d.GetOk("notify_no_data_timeframe"); ok {
		o.NoDataTimeframe = attr.(int)
	}
	if attr, ok := d.GetOk("renotify_interval"); ok {
		o.RenotifyInterval = attr.(int)
	}
	if attr, ok := d.GetOk("notify_audit"); ok {
		o.NotifyAudit = attr.(bool)
	}
	if attr, ok := d.GetOk("period"); ok {
		o.Period = attr.(int)
	}
	if attr, ok := d.GetOk("timeout_h"); ok {
		o.TimeoutH = attr.(int)
	}
	if attr, ok := d.GetOk("escalation_message"); ok {
		o.EscalationMessage = attr.(string)
	}

	m.Options = o

	if err := client.UpdateMonitor(m); err != nil {
		return fmt.Errorf("error updating monitor: %s", err.Error())
	}

	return resourceDatadogMonitorRead(d, meta)
}
