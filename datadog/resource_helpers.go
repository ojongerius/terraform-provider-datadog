package datadog

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
	"log"
	"strconv"
	"strings"
)

func thresholdSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"ok": &schema.Schema{
					Type:     schema.TypeFloat,
					Optional: true,
				},
				"warning": &schema.Schema{
					Type:     schema.TypeFloat,
					Optional: true,
				},
				"critical": &schema.Schema{
					Type:     schema.TypeFloat,
					Required: true,
				},
			},
		},
	}
}

func getThresholds(d *schema.ResourceData) (string, datadog.ThresholdCount) {
	t := datadog.ThresholdCount{}

	var threshold string

	if r, ok := d.GetOk("thresholds.ok"); ok {
		t.Ok = json.Number(r.(string))
	}

	if r, ok := d.GetOk("thresholds.warning"); ok {
		t.Warning = json.Number(r.(string))
	}

	if r, ok := d.GetOk("thresholds.critical"); ok {
		threshold = r.(string)
		t.Critical = json.Number(r.(string))
	}

	return threshold, t
}

func resourceDatadogGenericDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	if err = client.DeleteMonitor(i); err != nil {
		return err
	}

	return nil
}

func resourceDatadogGenericExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	// Exists - This is called to verify a resource still exists. It is called prior to Read,
	// and lowers the burden of Read to be able to assume the resource exists.
	client := meta.(*datadog.Client)

	// Workaround to handle upgrades from < 0.0.4
	if strings.Contains(d.Id(), "__") {
		return false, fmt.Errorf("Monitor ID contains __, which is pre v0.0.4 old behaviour.\n    You have the following options:\n" +
			"    * Run https://github.com/ojongerius/terraform-provider-datadog/blob/master/scripts/migration_helper.py to generate a new statefile and clean up monitors\n" +
			"    * Mannualy fix this by deleting all your metric_check resources and recreate them, " +
			"or manually remove half of the resources and hack the state file.\n")
	}

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, err
	}

	if _, err = client.GetMonitor(i); err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceDatadogGenericRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*datadog.Client)

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	r, err := client.GetMonitor(i)
	if err != nil {
		if strings.EqualFold(err.Error(), "API error 404 Not Found: {\"errors\":[\"Monitor not found\"]}") {
			return err
		}
		return err
	}

	// TODO: seriously consider how useful having a separate parser is after this resource has been simplified.
	m, err := resourceDatadogQueryParser(d, r)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] XX monitor: %v", m)
	if m.Name != "" {
		d.Set("name", m.Name)
	}
	if m.Message != "" {
		d.Set("message", m.Message)
	}
	if m.TimeAggregate != "" {
		d.Set("time_aggr", m.TimeAggregate)
	}
	if m.TimeWindow != "" {
		d.Set("time_windows", m.TimeWindow)
	}
	if m.SpaceAggregate != "" {
		d.Set("space_aggr", m.SpaceAggregate)
	}
	if m.Metric != "" {
		d.Set("metric", m.Metric)
	}

	if m.Operator != "" {
		d.Set("operator", m.Operator)
	}

	if m.Algorithm != "" {
		d.Set("algorithm", m.Algorithm)
	}

	if m.Check != "" {
		d.Set("check", m.Check)
	}

	// TODO: Double check this one
	if m.NotifyNoData != d.Get("notify_no_data") {
		d.Set("notify_no_data", m.NotifyNoData)
	}

	// TODO: Double check this one
	if m.NoDataTimeFrame != d.Get("no_data_timeframe") {
		d.Set("no_data_timeframe", m.NoDataTimeFrame)
	}

	if m.ReNotifyInterval != "" {
		d.Set("renotify_interval", m.ReNotifyInterval)
	}

	// TODO: test if these are not empty
	if len(m.Tags) > 0 {
		log.Printf("[DEBUG] XX tags were found, setting state for diff")
		d.Set("tags", m.Tags)
	}

	if len(m.Keys) > 0 {
		log.Printf("[DEBUG] XX keys were found, setting state for diff")
		d.Set("keys", m.Keys)
	}

	if m.Threshold != "" {
		d.Set("threshold", m.Threshold)
	}

	return nil
}

func monitorCreator(d *schema.ResourceData, meta interface{}, m *datadog.Monitor) error {
	client := meta.(*datadog.Client)

	m, err := client.CreateMonitor(m)
	if err != nil {
		return fmt.Errorf("error updating montor: %s", err.Error())
	}

	d.SetId(strconv.Itoa(m.Id))

	return nil
}

func monitorUpdater(d *schema.ResourceData, meta interface{}, m *datadog.Monitor) error {
	client := meta.(*datadog.Client)

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	m.Id = i

	if err = client.UpdateMonitor(m); err != nil {
		return fmt.Errorf("error updating montor: %s", err.Error())
	}

	return nil
}
