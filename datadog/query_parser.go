package datadog

import (
	"errors"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
	"log"
	"regexp"
	"strings"
)

// resourceDatadogQueryParser takes d, with resource data, m containing a monitoring and resourceType a string with the resource name/type.
func resourceDatadogQueryParser(d *schema.ResourceData, m *datadog.Monitor, resourceType string) error {

	/*
		This is the first iteration of of a generic query parser. It saves state too, should it be renamed?

	*/

	// Name -this is identical across resources.
	re := regexp.MustCompile(`\[([a-zA-Z]+)\]\s(.+)`)
	r := re.FindStringSubmatch(m.Name) // Find check name
	if r == nil {
		return errors.New("Name parser error: string match returned nil")
	}
	if len(r) < 3 {
		return errors.New(fmt.Sprintf("Name parser error. Expected: 3. Got: %d", len(r)))
	}
	level := r[1] // Store this so we can save the contact for in the right place (see below)
	log.Printf("[DEBUG] found level %s", level)
	log.Printf("[DEBUG] found name %s", r[2])
	d.Set("name", r[2])

	// Message -this would be identical across resources too
	res := strings.Split(m.Message, " @") // TODO: use must compile for this one
	if res == nil {
		return errors.New("Message parser error: string split returned nil")
	}

	log.Printf("[DEBUG] found message %s", res[0])
	d.Set("message", res[0])
	for k, v := range res {
		if k == 0 {
			// The message is the first element, move on to the contact
			// TODO: handle cases where at-mentions are embedded/nested *in* the messages.
			continue
		}
		log.Printf("[DEBUG] found %s.notify: %s", level, v)
		d.Set(fmt.Sprintf("%s.notify", level), v)
	}

	// Query -this needs to receive (a) pattern(s) for each resource. AFAIK the only (considerable) different
	// resource would be Outliers. TODO: add logic to use regexps per type. Map makes sense.
	re_test_multi := regexp.MustCompile(`by {`)
	result := re_test_multi.MatchString(m.Query)
	if result {
		log.Print("[DEBUG] Found multi alert")
		re = regexp.MustCompile(`(?P<time_aggr>[\w]{3}?)\((?P<time_window>[a-zA-Z0-9_]+?)\):(?P<space_aggr>[a-zA-Z]+?):(?P<metric>[_.a-zA-Z0-9]+){(?P<tags>[a-zA-Z0-9_:*]+?)}\s+by\s+{(?P<keys>[a-zA-Z0-9_*]+?)}\s+(?P<operator>[><=]+?)\s+(?P<threshold>[0-9]+)`)
		// TODO: ^^ Break this up in multiple constants
	} else {
		log.Print("[DEBUG] Found simple alert")
		re = regexp.MustCompile(`(?P<time_aggr>[\w]{3}?)\((?P<time_window>[a-zA-Z0-9_]+?)\):(?P<space_aggr>[a-zA-Z]+?):(?P<metric>[_.a-zA-Z0-9]+){(?P<tags>[a-zA-Z0-9_:*]+?)}\s+(?P<operator>[><=]+?)\s+(?P<threshold>[0-9]+)`)
		// TODO: ^^ Break this up in multiple constants
	}
	n1 := re.SubexpNames()
	subMatches := re.FindAllStringSubmatch(m.Query, -1)
	log.Printf("[DEBUG] Submatches: %v", subMatches)
	for k, _ := range n1 {
		if k > (len(subMatches) - 1) {
			continue
		}
		// TODO: Find a way to generate, or let the caller specify the list
		r2 := subMatches[k]
		for i, n := range r2 {
			if n != "" {
				switch {
				case n1[i] == "time_aggr": // Shared
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("time_aggr", n)
				case n1[i] == "time_window": // Shared
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("time_window", n)
				case n1[i] == "space_aggr": // Shared
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("space_aggr", n)
				case n1[i] == "metric": // Shared
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("metric", n)
				case n1[i] == "tags": // Shared
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("tags", n)
				case n1[i] == "keys": // Shared
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("keys", n)
				case n1[i] == "operator": // Shared
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("operator", n)
				case n1[i] == "threshold": // Shared
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set(fmt.Sprintf("%s.threshold", level), n)
				case n1[i] == "algorithm": // Outlier resource
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("algorithm", n)
				case n1[i] == "check": // Check resource
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("check", n)
				case n1[i] == "renotify_interval": // Check resource
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("renotify_interval", n)
				}
			}
		}

	}
	log.Printf("[DEBUG] storing  %v", m.Options.NotifyNoData)
	d.Set("notify_no_data", m.Options.NotifyNoData)
	log.Printf("[DEBUG] storing  %v", m.Options.NoDataTimeframe)
	d.Set("no_data_timeframe", m.Options.NoDataTimeframe)

	return nil
}
