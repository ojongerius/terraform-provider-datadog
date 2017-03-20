package datadog

import (
	"bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

// resourceDatadogMetricAlert is a Datadog monitor resource
func resourceDatadogMetricAlert() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogMetricAlertCreate,
		Read:   resourceDatadogGenericRead,
		Update: resourceDatadogMetricAlertUpdate,
		Delete: resourceDatadogGenericDelete,
		Exists: resourceDatadogGenericExists,

		Schema: map[string]*schema.Schema{
			// Common
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"message": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"query": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

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
			// TODO: common options that have not been implemented, and should be there for the other
			// monitors too.
			/*

				silenced dictionary of scopes to timestamps or None. Each scope will be muted until the given POSIX timestamp or forever if the value is None.
				Default: None

				Examples:

				To mute the alert completely:
				{'*': None}

				To mute role:db for a short time:
				{'role:db': 1412798116}

				timeout_h the number of hours of the monitor not reporting data before it will automatically resolve from a triggered state.
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

			*/

			// Specific
			"metric": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"query"},
			},
		},
	}
}

// buildMonitorStruct returns a monitor struct
func buildMetricAlertStruct(d *schema.ResourceData) *datadog.Monitor {
	name := d.Get("name").(string)
	message := d.Get("message").(string)
	timeAggr := d.Get("time_aggr").(string)
	timeWindow := d.Get("time_window").(string)
	spaceAggr := d.Get("space_aggr").(string)
	metric := d.Get("metric").(string)
	query := d.Get("query").(string)

	// Tags are are no separate resource/gettable, so some trickery is needed
	var buffer bytes.Buffer
	if raw, ok := d.GetOk("tags"); ok {
		list := raw.([]interface{})
		length := (len(list) - 1)
		for i, v := range list {
			buffer.WriteString(fmt.Sprintf("%s", v))
			if i != length {
				buffer.WriteString(",")
			}

		}
	}

	tagsParsed := buffer.String()

	// Keys are used for multi alerts
	var b bytes.Buffer
	if raw, ok := d.GetOk("keys"); ok {
		list := raw.([]interface{})
		b.WriteString("by {")
		length := (len(list) - 1)
		for i, v := range list {
			b.WriteString(fmt.Sprintf("%s", v))
			if i != length {
				b.WriteString(",")
			}

		}
		b.WriteString("}")
	}

	keys := b.String()

	threshold, thresholds := getThresholds(d)

	operator := d.Get("operator").(string)

	var q string

	if query == "" {
		q = fmt.Sprintf("%s(%s):%s:%s{%s} %s %s %s", timeAggr,
			timeWindow,
			spaceAggr,
			metric,
			tagsParsed,
			keys,
			operator,
			threshold)
	} else {
		q = fmt.Sprintf("%s %s %s", query, operator, threshold)
	}

	log.Print(fmt.Sprintf("[DEBUG] submitting query: %s", q))

	o := datadog.Options{
		NotifyNoData:     d.Get("notify_no_data").(bool),
		NoDataTimeframe:  d.Get("no_data_timeframe").(int),
		RenotifyInterval: d.Get("renotify_interval").(int),
		Thresholds:       thresholds,
	}

	m := datadog.Monitor{
		Type:    "metric alert",
		Query:   q,
		Name:    name,
		Message: message,
		Options: o,
	}

	return &m
}

func resourceDatadogMetricAlertCreate(d *schema.ResourceData, meta interface{}) error {

	m := buildMetricAlertStruct(d)
	if err := monitorCreator(d, meta, m); err != nil {
		return err
	}

	return nil
}

func resourceDatadogMetricAlertUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] running update.")

	m := buildMetricAlertStruct(d)
	if err := monitorUpdater(d, meta, m); err != nil {
		return err
	}

	return nil
}
