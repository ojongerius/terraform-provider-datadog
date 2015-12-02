package datadog

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
	"log"
	"regexp"
	"strings"
)

/*
	Example of simple query:
	"min(last_15m):avg:stats.aws.vpc_prod_monitoring.ami.count{service_name:gnomes,aws-account-alias:vpc-production} > 3500"

	Example of multi alert query:
	"avg(last_15m):avg:system.disk.in_use{*} by {host,device} > 0.9"

    Outliers query (inherently a multi alert):
	"avg(last_1h):outliers(avg:system.fs.inodes.in_use{*} by {host},'dbscan',2) > 0"
*/

const (
	timeAggrRegexp   = "(?P<time_aggr>[\\w]{3}?)"
	timeWinRegexp    = "(?P<time_window>[a-zA-Z0-9_]+?)"
	spaceAggrRegexp  = "(?P<space_aggr>[a-zA-Z]+?)"
	metricRegexp     = "(?P<metric>[_.a-zA-Z0-9]+)"
	tagsRegexp       = "{(?P<tags>[a-zA-Z0-9_:*,-]+?)}"
	baseRegexp       = timeAggrRegexp + "\\(" + timeWinRegexp + "\\):" + spaceAggrRegexp + ":" + metricRegexp + tagsRegexp
	conditionRegexp  = "\\s+(?P<operator>[><=]+?)\\s+(?P<threshold>[0-9]+)"
	multiAlertRegexp = "\\s+by\\s+{(?P<keys>[a-zA-Z0-9_*,-]+?)}"
	algorithmRegexp  = "'(?P<algorithm>[a-zA-Z]+)'"
	thresholdRegexp  = "(?P<threshold>[0-9]+)"
	outlierRegexp    = timeAggrRegexp + "\\(" + timeWinRegexp + "\\):outliers\\(" + spaceAggrRegexp + ":" + metricRegexp + tagsRegexp + multiAlertRegexp + "," + algorithmRegexp + "," + thresholdRegexp + "\\)"
)

func resourceDatadogQueryParser(d *schema.ResourceData, m *datadog.Monitor) error {
	// Name
	re := regexp.MustCompile(`\[([a-zA-Z]+)\]\s(.+)`)
	r := re.FindStringSubmatch(m.Name) // Find check name
	if r == nil {
		return fmt.Errorf("Name parser error: string match returned nil")
	}
	if len(r) < 3 {
		return fmt.Errorf("Name parser error. Expected: 3. Got: %d", len(r))
	}
	level := r[1] // Store this so we can save the contact for in the right place (see below)
	log.Printf("[DEBUG] found level %s", level)
	log.Printf("[DEBUG] found name %s", r[2])
	d.Set("name", r[2])

	// Message
	res := strings.Split(m.Message, " @")
	if res == nil {
		return fmt.Errorf("Message parser error: string split returned nil")
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

	// If it is an outlier, use separate regular expression. Outliers can only be grouped, and hence alway are multi alerts.
	if strings.Contains(m.Query, "outliers") {
		log.Print("[DEBUG] is Outlier alert")
		re = regexp.MustCompile(outlierRegexp + conditionRegexp)
		log.Printf("[DEBUG] setting regexp to: %s", outlierRegexp+conditionRegexp)
	} else {
		// No outlier? Test if it is a simple or a "multi alert" monitor
		reTestMulti := regexp.MustCompile(`by {`)
		result := reTestMulti.MatchString(m.Query)
		if result {
			log.Print("[DEBUG] Found multi alert")
			re = regexp.MustCompile(baseRegexp + multiAlertRegexp + conditionRegexp)
			log.Printf("[DEBUG] setting regexp to: %s", baseRegexp+multiAlertRegexp+conditionRegexp)
		} else {
			log.Print("[DEBUG] Found simple alert")
			re = regexp.MustCompile(baseRegexp + conditionRegexp)
			log.Printf("[DEBUG] setting regexp to: %s", baseRegexp+conditionRegexp)
		}
	}
	n1 := re.SubexpNames()
	log.Printf("[DEBUG] query: %s", m.Query)
	subMatches := re.FindAllStringSubmatch(m.Query, -1)
	log.Printf("[DEBUG] submatches: %v", subMatches)
	for k := range n1 {
		if k > (len(subMatches) - 1) {
			continue
		}
		// TODO: Find a way to generate, or let the caller specify the list
		r2 := subMatches[k]
		for i, n := range r2 {
			if n != "" {
				switch {
				case n1[i] == "time_aggr": // Shared
					log.Printf("[DEBUG] storing time_aggr: %s", n1[i])
					d.Set("time_aggr", n)
				case n1[i] == "time_window": // Shared
					log.Printf("[DEBUG] storing time_window: %s", n1[i])
					d.Set("time_window", n)
				case n1[i] == "space_aggr": // Shared
					log.Printf("[DEBUG] storing  space_aggr: %s", n1[i])
					d.Set("space_aggr", n)
				case n1[i] == "metric": // Shared
					log.Printf("[DEBUG] storing metric: %s", n1[i])
					d.Set("metric", n)
				case n1[i] == "tags": // Shared
					log.Printf("[DEBUG] storing tags: %s", n1[i])
					d.Set("tags", n)
				case n1[i] == "keys": // Shared
					log.Printf("[DEBUG] storing keys: %s", n1[i])
					d.Set("keys", n)
				case n1[i] == "operator": // Shared
					log.Printf("[DEBUG] storing operator: %s", n1[i])
					d.Set("operator", n)
				case n1[i] == "threshold": // Shared
					log.Printf("[DEBUG] storing threshold: %s", n1[i])
					d.Set(fmt.Sprintf("%s.threshold", level), n)
				case n1[i] == "algorithm": // Outlier resource
					log.Printf("[DEBUG] storing algorithm: %s", n1[i])
					d.Set("algorithm", n)
				case n1[i] == "check": // Check resource
					log.Printf("[DEBUG] storing check: %s", n1[i])
					d.Set("check", n)
				case n1[i] == "renotify_interval": // Check resource
					log.Printf("[DEBUG] storing renotify_interval: %s", n1[i])
					d.Set("renotify_interval", n)
				}
			}
		}

	}
	log.Printf("[DEBUG] storing notify_no_data: %v", m.Options.NotifyNoData)
	d.Set("notify_no_data", m.Options.NotifyNoData)
	log.Printf("[DEBUG] storing nodata_time_frame: %v", m.Options.NoDataTimeframe)
	d.Set("no_data_timeframe", m.Options.NoDataTimeframe)

	return nil
}
