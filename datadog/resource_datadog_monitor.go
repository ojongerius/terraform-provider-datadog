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

			// TODO figure out how to merge this
			"thresholds": thresholdSchema(),

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
			// TODO: add all the other options that are possible
			// TODO: can make some options exclusive, see last merge to master
			/*
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

				notify_no_data a boolean indicating whether this monitor will notify when data stops reporting.
				Default: false

				no_data_timeframe the number of minutes before a monitor will notify when data stops reporting. Must be at least 2x the monitor timeframe for metric alerts or 2 minutes for service checks.
				Default: 2x timeframe for metric alerts, 2 minutes for service checks

				timeout_h the number of hours of the monitor not reporting data before it will automatically resolve from a triggered state.
				Default: None

				renotify_interval the number of minutes after the last notification before a monitor will re-notify on the current status. It will only re-notify if it's not resolved.
				Default: None

				escalation_message a message to include with a re-notification. Supports the '@username' notification we allow elsewhere. Not applicable if renotify_interval is None.
				Default: None

				notify_audit a boolean indicating whether tagged users will be notified on changes to this monitor.
				Default: False

				include_tags a boolean indicating whether notifications from this monitor will automatically insert its triggering tags into the title.
				Default: True

				Examples:

				True:
				[Triggered on {host:h1}] Monitor Title

				False:
				[Triggered] Monitor Title

				METRIC ALERT OPTIONS

				These options only apply to metric alerts.
				thresholds a dictionary of thresholds by threshold type. Currently we have two threshold types for metric alerts: critical and warning. Critical is defined in the query, but can also be specified in this option. Warning threshold can only be specified using the thresholds option.
				Example: {'critical': 90, 'warning': 80}

				SERVICE CHECK OPTIONS

				These options only apply to service checks and will be ignored for other monitor types.
				thresholds a dictionary of thresholds by status. Because service checks can have multiple thresholds, we don't define them directly in the query.
				Default: {'ok': 1, 'critical': 1, 'warning': 1}

			*/
		},
	}
}

// buildMonitorStruct returns a monitor struct
func buildMonitorStruct(d *schema.ResourceData) *datadog.Monitor {

	_, thresholds := getThresholds(d)

	o := datadog.Options{
		NotifyNoData:     d.Get("notify_no_data").(bool),
		NoDataTimeframe:  d.Get("no_data_timeframe").(int),
		RenotifyInterval: d.Get("renotify_interval").(int),
		Thresholds:       thresholds,
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
	d.Set("thresholds", m.Options.Thresholds) // would this work?
	d.Set("notify_no_data", m.Options.NotifyNoData)
	d.Set("notify_no_data_timeframe", m.Options.NoDataTimeframe)
	d.Set("renotify_interval", m.Options.RenotifyInterval)

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

	m.Options = o

	if err := client.UpdateMonitor(m); err != nil {
		return fmt.Errorf("error updating monitor: %s", err.Error())
	}

	return resourceDatadogMonitorRead(d, meta)
}
