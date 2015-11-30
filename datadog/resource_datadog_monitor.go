package datadog

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/zorkian/go-datadog-api"

	"github.com/hashicorp/terraform/helper/schema"
)

// resourceDatadogMonitor is a Datadog monitor resource
func resourceDatadogMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogMonitorCreate,
		Read:   resourceDatadogMonitorRead,
		Update: resourceDatadogMonitorUpdate,
		Delete: resourceDatadogMonitorDelete,
		Exists: resourceDatadogGenericExists,

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

// buildMonitorStruct returns a monitor struct
func buildMonitorStruct(d *schema.ResourceData, typeStr string) *datadog.Monitor {
	// TODO: Add support for service checks
	/*
			For service checks, query should be:

			"check".over(tags).last(count).count_by_status()
				* "check" name of the check, e.g. datadog.agent.up
			    * "tags" one or more quoted tags (comma-separated), or "*". e.g.: .over("env:prod", "role:db")
				* "count" must be at >= your max threshold (defined in the options). e.g. if you want to notify on 1 critical, 3 ok and 2 warn statuses count should be 3.

		    For metric checks:

			* time_aggr(time_window):space_aggr:metric{tags} [by {key}] operator #
			* time_aggr avg, sum, max, min, change, or pct_change
			* time_window last_#m (5, 10, 15, or 30), last_#h (1, 2, or 4), or last_1d
			* space_aggr avg, sum, min, or max
			* tags one or more tags (comma-separated), or *
			* key a 'key' in key:value tag syntax; defines a separate alert for each tag in the group (multi-alert)
			* operator <, <=, >, >=, ==, or !=
			* # an integer or decimal number used to set the threshold
			If you are using the change or pct_change time aggregator, you can instead use change_aggr(time_aggr(time_window), timeshift):space_aggr:metric{tags} [by {key}] operator # with:
			* change_aggr change, pct_change
			* time_aggr avg, sum, max, min
			* time_window last_#m (1, 5, 10, 15, or 30), last_#h (1, 2, or 4), or last_#d (1 or 2)
			* timeshift #m_ago (5, 10, 15, or 30), #h_ago (1, 2, or 4), or 1d_ago
	*/
	name := d.Get("name").(string)
	message := d.Get("message").(string)
	timeAggr := d.Get("time_aggr").(string)
	timeWindow := d.Get("time_window").(string)
	spaceAggr := d.Get("space_aggr").(string)
	metric := d.Get("metric").(string)
	tags := d.Get("metric_tags").(string)
	operator := d.Get("operator").(string)
	query := fmt.Sprintf("%s(%s):%s:%s{%s} %s %s", timeAggr,
		timeWindow,
		spaceAggr,
		metric,
		tags,
		operator,
		d.Get(fmt.Sprintf("%s.threshold", typeStr)))

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

// resourceDatadogMonitorCreate creates a monitor.
func resourceDatadogMonitorCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	w, err := client.CreateMonitor(buildMonitorStruct(d, "warning"))

	if err != nil {
		return fmt.Errorf("error creating warning: %s", err)
	}

	c, cErr := client.CreateMonitor(buildMonitorStruct(d, "critical"))

	if cErr != nil {
		return fmt.Errorf("error creating warning: %s", cErr)
	}

	log.Printf("[DEBUG] Saving IDs: %s__%s", strconv.Itoa(w.Id), strconv.Itoa(c.Id))

	d.SetId(fmt.Sprintf("%s__%s", strconv.Itoa(w.Id), strconv.Itoa(c.Id)))

	return nil
}

// resourceDatadogMonitorDelete deletes a monitor.
func resourceDatadogMonitorDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	for _, v := range strings.Split(d.Id(), "__") {
		if v == "" {
			return fmt.Errorf("Id not set.")
		}
		ID, iErr := strconv.Atoi(v)

		if iErr != nil {
			return iErr
		}

		err := client.DeleteMonitor(ID)
		if err != nil {
			return err
		}
	}
	return nil
}

// resourceDatadogMonitorRead synchronises Datadog and local state .
func resourceDatadogMonitorRead(d *schema.ResourceData, meta interface{}) error {
	// TODO: add support for this a read function.
	/* Read - This is called to resync the local state with the remote state.
	Terraform guarantees that an existing ID will be set. This ID should be
	used to look up the resource. Any remote data should be updated into the
	local data. No changes to the remote resource are to be made.
	*/

	return nil
}

// resourceDatadogMonitorUpdate updates a monitor.
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

	warningID, iErr := strconv.Atoi(wID)

	if iErr != nil {
		return iErr
	}

	criticalID, iErr := strconv.Atoi(cID)

	if iErr != nil {
		return iErr
	}

	client := meta.(*datadog.Client)

	warningBody := buildMonitorStruct(d, "warning")
	criticalBody := buildMonitorStruct(d, "critical")

	warningBody.Id = warningID
	criticalBody.Id = criticalID

	wErr := client.UpdateMonitor(warningBody)

	if wErr != nil {
		return fmt.Errorf("error updating warning: %s", wErr.Error())
	}

	cErr := client.UpdateMonitor(criticalBody)

	if cErr != nil {
		return fmt.Errorf("error updating critical: %s", cErr.Error())
	}

	return nil
}
