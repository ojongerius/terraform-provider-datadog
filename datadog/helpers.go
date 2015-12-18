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

	// TODO clean up
	levels := []string{"warning", "critical"}

	for i, v := range strings.Split(d.Id(), "__") {
		if v == "" {
			return fmt.Errorf("Id not set.")
		}
		ID, err := strconv.Atoi(v)
		if err != nil {
			return err
		}

		m, err := client.GetMonitor(ID)
		if err != nil {
			if strings.EqualFold(err.Error(), "API error 404 Not Found: {\"errors\":[\"Monitor not found\"]}") {
				log.Printf("[DEBUG] XX marking monitor %s a not existing: %s", v, err)
				// TODO: vanishing should trigger recreation. How do we get this done with? We can't fail,
				// if we set it to be an empty monitor object, that would not trigger anything.
				// Ideas to explore:
				// * 1 Save the IDs in say int warning.ID (and set it to something non default (0) and not possible
				//    ^^ this is kind of nice, as if it vanished we set it to "", but how would that work
				//            diff wise? Shall we just keep it in the contrived ID?
				// * 2 Save to bool warning.exists (but maybe inverse warning.vanished because default is false)
				//    ^^ this is better as option 3, and the default value could be true.
				// * 3 Save it to a string so we don't have the fucking default value issue
				//monitors[i].Recreate = true
				//    ^^ sure this worked, but we now have a kind of useless thing.
				//d.Set(fmt.Sprintf("%s.exists", levels[i]), true)
				// Set the ID here
				// TODO improve this
				levelMap := d.Get(levels[i]).(map[string]interface{})
				levelMap["id"] = ""
				if err := d.Set(levels[i], levelMap); err != nil {
					return err
				}
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
	for i, m := range monitors {
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

		levelMap := d.Get(levels[i]).(map[string]interface{})
		if m.Threshold != "" {
			levelMap["threshold"] = m.Threshold
			log.Printf("XX threshold %s", m.Threshold)
		} else {
			log.Printf("XX threshold not found")
			log.Printf("Object: %v", d.Get(fmt.Sprintf(levels[i])))
		}
		if m.Notify != "" {
			log.Printf("XX notify %s", m.Notify)
			levelMap["notify"] = m.Notify
		} else {
			log.Printf("XX notify not found")
		}
		err := d.Set(levels[i], levelMap)
		if err != nil {
			log.Printf("XX error writing values! %s", err)
		}
		log.Printf("REsulting Object: %v", d.Get(fmt.Sprintf(levels[i])))
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

	// TODO stop abusing this
	levels := []string{"warning", "critical"}

	// Set default to false
	exists := false
	for i, v := range strings.Split(d.Id(), "__") {
		if v == "" {
			log.Printf("[DEBUG] Could not parse IDs: %s", v)
			return false, fmt.Errorf("Id not set.")
		}
		ID, err := strconv.Atoi(v)
		if err != nil {
			log.Printf("[DEBUG] Received error converting string: %s", err)
			return false, err
		}

		if _, err = client.GetMonitor(ID); err != nil {
			if strings.EqualFold(err.Error(), "API error 404 Not Found: {\"errors\":[\"Monitor not found\"]}") {
				log.Printf("[DEBUG] monitor %s does not exist: %s", v, err)
				levelMap := make(map[string]string)
				levelMap["id"] = ""
				if err := d.Set(fmt.Sprintf(levels[i]), levelMap); err != nil {
					return false, err
				}
				continue
			}
			log.Printf("[DEBUG] received error getting monitor %s: %s", v, err)
			return false, err
		}
		log.Printf("[DEBUG] found monitor %s", v)
		exists = true
	}

	return exists, e
}
