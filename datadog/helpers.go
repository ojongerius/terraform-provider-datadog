package datadog

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
	"log"
	"strconv"
	"strings"
)

func resourceDatadogGenericRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*datadog.Client)
	monitors := make([]subDatadogMonitor, 2)

	for i, v := range strings.Split(d.Id(), "__") {
		if v == "" {
			return fmt.Errorf("Id not set.")
		}
		ID, iErr := strconv.Atoi(v)

		if iErr != nil {
			return iErr
		}

		m, err := client.GetMonitor(ID)

		if err != nil {
			if strings.EqualFold(err.Error(), "API error 404 Not Found: {\"errors\":[\"Monitor not found\"]}") {
				log.Printf("[DEBUG] XX marking monitor %s a not existing: %s", v, err)
				// TODO: vanishing should trigger recreation. How do we get this done with? We can't fail,
				// if we set it to be an empty monitor object, that would not trigger anything.
				// Ideas to explore:
				// * Save the IDs in say int warning.ID (and set it to something non default (0) and not possible
				// * Save to bool warning.exists (but maybe inverse warning.vanished because default is false)
				// * Save it to a string so we don't have the fucking default value issue
				//monitors[i].Recreate = true
				d.Set("recreate", true)
				continue
			}
			return err
		}

		monitors[i], err = resourceDatadogQueryParser(d, m)
		if err != nil {
			return err
		}

	}

	log.Printf("[DEBUG] XX amount of monitors: %v", len(monitors))

	// TODO: Better, less tedious way to do this?
	for _, m := range monitors {
		log.Printf("[DEBUG] XX monitor: %v", m)
		if m.Name != "" {
			d.Set("name", m.Name)
		}
		if m.Message != "" {
			d.Set("message", m.Message)
		}
		if m.Notify != "" {
			d.Set("notify", m.Notify)
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

		if m.Threshold != "" {
			d.Set("threshold", m.Threshold)
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
	}

	return nil
}

func resourceDatadogGenericExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	// Exists - This is called to verify a resource still exists. It is called prior to Read,
	// and lowers the burden of Read to be able to assume the resource exists.
	//
	// If the resource is no longer present in remote state, calling SetId with an empty string will
	// signal its removal.

	client := meta.(*datadog.Client)

	// Set default to false
	exists := false
	for _, v := range strings.Split(d.Id(), "__") {
		if v == "" {
			log.Printf("[DEBUG] Could not parse IDs: %s", v)
			return false, fmt.Errorf("Id not set.")
		}
		ID, iErr := strconv.Atoi(v)

		if iErr != nil {
			log.Printf("[DEBUG] Received error converting string: %s", iErr)
			return false, iErr
		}
		_, err := client.GetMonitor(ID)
		if err != nil {

			if strings.EqualFold(err.Error(), "API error 404 Not Found: {\"errors\":[\"Monitor not found\"]}") {
				log.Printf("[DEBUG] monitor %s does not exist: %s", v, err)
				continue
			} else {
				log.Printf("[DEBUG] received error getting monitor %s: %s", v, err)
				e = err
				continue
			}
		}
		log.Printf("[DEBUG] found monitor %s", v)
		exists = true
	}

	return exists, e
}
