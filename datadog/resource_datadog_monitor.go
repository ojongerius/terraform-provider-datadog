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
			"escalation_message": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"query": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			/*
				"tags": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			*/

			// Options
			"thresholds": thresholdSchema(),
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
			},
			"notify_audit": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"timeout_h": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			// TODO should actually be map[string]int
			"silenced": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					Elem: &schema.Schema{
						Type: schema.TypeInt},
				},
			},
			"include_tags": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

// buildMonitorStruct returns a monitor struct
func buildMonitorStruct(d *schema.ResourceData) *datadog.Monitor {

	_, thresholds := getThresholds(d)

	o := datadog.Options{
		Thresholds: thresholds,
	}
	if attr, ok := d.GetOk("silenced"); ok {
		s := make(map[string]int)
		// TODO: this is not very defensive, test if we can fail non int input
		for k, v := range attr.(map[string]interface{}) {
			s[k], _ = strconv.Atoi(v.(string))
		}
		o.Silenced = s
	}
	if attr, ok := d.GetOk("notify_data"); ok {
		o.NotifyNoData = attr.(bool)
	}
	if attr, ok := d.GetOk("no_data_timeframe"); ok {
		o.NoDataTimeframe = attr.(int)
	}
	if attr, ok := d.GetOk("renotify_interval"); ok {
		o.RenotifyInterval = attr.(int)
	}
	if attr, ok := d.GetOk("notify_audit"); ok {
		o.NotifyAudit = attr.(bool)
	}
	if attr, ok := d.GetOk("timeout_h"); ok {
		o.TimeoutH = attr.(int)
	}
	if attr, ok := d.GetOk("escalation_message"); ok {
		o.EscalationMessage = attr.(string)
	}
	if attr, ok := d.GetOk("escalation_message"); ok {
		o.EscalationMessage = attr.(string)
	}
	if attr, ok := d.GetOk("include_tags"); ok {
		o.IncludeTags = attr.(bool)
	}

	m := datadog.Monitor{
		Type:    d.Get("type").(string),
		Query:   d.Get("query").(string),
		Name:    d.Get("name").(string),
		Message: d.Get("message").(string),
		Options: o,
	}

	/*
		if attr, ok := d.GetOk("tags"); ok {
			m.Tags = attr.([]string)
		}
	*/

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
	/*
		d.Set("tags", m.Tags)
	*/
	d.Set("thresholds", m.Options.Thresholds)
	d.Set("notify_no_data", m.Options.NotifyNoData)
	d.Set("notify_no_data_timeframe", m.Options.NoDataTimeframe)
	d.Set("renotify_interval", m.Options.RenotifyInterval)
	d.Set("notify_audit", m.Options.NotifyAudit)
	d.Set("timeout_h", m.Options.TimeoutH)
	d.Set("escalation_message", m.Options.EscalationMessage)
	d.Set("silenced", m.Options.Silenced)
	d.Set("include_tags", m.Options.IncludeTags)

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
	/*
		if attr, ok := d.GetOk("tags"); ok {
			m.Tags = attr.([]string)
		}
	*/

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
	if attr, ok := d.GetOk("timeout_h"); ok {
		o.TimeoutH = attr.(int)
	}
	if attr, ok := d.GetOk("escalation_message"); ok {
		o.EscalationMessage = attr.(string)
	}
	if attr, ok := d.GetOk("silenced"); ok {
		// TODO: this is not very defensive, test if we can fail non int input
		s := make(map[string]int)
		for k, v := range attr.(map[string]interface{}) {
			s[k], _ = strconv.Atoi(v.(string))
		}
		o.Silenced = s
	}
	if attr, ok := d.GetOk("include_tags"); ok {
		o.IncludeTags = attr.(bool)
	}

	m.Options = o

	if err := client.UpdateMonitor(m); err != nil {
		return fmt.Errorf("error updating monitor: %s", err.Error())
	}

	return resourceDatadogMonitorRead(d, meta)
}
