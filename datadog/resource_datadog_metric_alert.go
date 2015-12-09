package datadog

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/zorkian/go-datadog-api"

	"github.com/hashicorp/terraform/helper/schema"
)

// resourceDatadogMetricAlert is a Datadog monitor resource
func resourceDatadogMetricAlert() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogMetricAlertCreate,
		Read:   resourceDatadogGenericRead,
		Update: resourceDatadogMetricAlertUpdate,
		Delete: resourceDatadogMetricAlertDelete,
		Exists: resourceDatadogGenericExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"metric": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"keys": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
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

			// TODO: too evil? Add to other resources too. Should we "hide" in meta?
			"recreate": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
		},
	}
}

// buildMonitorStruct returns a monitor struct
func buildMetricAlertStruct(d *schema.ResourceData, typeStr string) *datadog.Monitor {
	name := d.Get("name").(string)
	message := d.Get("message").(string)
	timeAggr := d.Get("time_aggr").(string)
	timeWindow := d.Get("time_window").(string)
	spaceAggr := d.Get("space_aggr").(string)
	metric := d.Get("metric").(string)

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

	operator := d.Get("operator").(string)
	query := fmt.Sprintf("%s(%s):%s:%s{%s} %s %s %s", timeAggr,
		timeWindow,
		spaceAggr,
		metric,
		tagsParsed,
		keys,
		operator,
		d.Get(fmt.Sprintf("%s.threshold", typeStr)))

	log.Print(fmt.Sprintf("[DEBUG] submitting query: %s", query))

	o := datadog.Options{
		NotifyNoData:    d.Get("notify_no_data").(bool),
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

func resourceDatadogMetricAlertCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	// TODO: refactor out into generic create function that only takes the specific
	// struct. After wire duplexer.
	levels := []string{"warning", "critical"}
	ids := make([]int, len(levels))

	for i, l := range levels {
		m, err := client.CreateMonitor(buildMetricAlertStruct(d, l))
		ids[i] = m.Id
		if err != nil {
			return fmt.Errorf("error creating %s: %s", l, err)
		}
	}

	log.Printf("[DEBUG] Saving IDs: %s__%s", strconv.Itoa(ids[0]), strconv.Itoa(ids[0]))
	d.SetId(fmt.Sprintf("%s__%s", strconv.Itoa(ids[0]), strconv.Itoa(ids[1])))

	return nil
}

func resourceDatadogMetricAlertDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	// TODO: refactor, wire duplexer in

	for _, v := range strings.Split(d.Id(), "__") {
		if v == "" {
			return fmt.Errorf("Id not set.")
		}
		ID, err := strconv.Atoi(v)
		if err != nil {
			return err
		}

		if err = client.DeleteMonitor(ID); err != nil {
			return err
		}
	}
	return nil
}

func resourceDatadogMetricAlertUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] running update.")
	// TODO: refactor to call the "duplexer" instead of all this split bullshit

	client := meta.(*datadog.Client)

	levels := []string{"warning", "critical"}
	ids := make([]int, len(levels))

	for i, v := range strings.Split(d.Id(), "__") {
		if v == "" {
			return fmt.Errorf("Id not set.")
		}

		// Build monitor body for our level
		monitorBody := buildMetricAlertStruct(d, levels[i])

		var err error
		monitorBody.Id, err = strconv.Atoi(v)
		if err != nil {
			return err
		}

		// Save body to update state in the end
		ids[i] = monitorBody.Id
		// Update monitor, 404 implies our monitor may have been manually deleted, and
		// attempt to create it.
		if err = client.UpdateMonitor(monitorBody); err != nil {
			if strings.EqualFold(err.Error(), "API error 404 Not Found: {\"errors\":[\"Monitor not found\"]}") {
				// TODO: remove XX when done log.
				log.Printf("[DEBUG] XX monitor does not exist, recreating")
				m, err := client.CreateMonitor(monitorBody)
				if err != nil {
					return fmt.Errorf("error creating warning: %s", err.Error())
				}
				// This is our new ID
				ids[i] = m.Id
			}
			return fmt.Errorf("error updating warning: %s", err.Error())
		}
	}

	d.SetId(fmt.Sprintf("%d__%d", ids[0], ids[1]))

	// After an update we can "unset" recreate. log.Printf("[DEBUG] XX Unsetting recreate")FJJJJ
	d.Set("recreate", false)

	return nil
}
