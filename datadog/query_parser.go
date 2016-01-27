package datadog

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
	"log"
	"reflect"
	"regexp"
	"strings"
)

/*
    TODO: summary: consider getting rid of this whole thing and just be heaps closer to the API.

    OR: as we now just have one monitor per resource, and no longer have
        to parse notify from message, we *could* support constructing the
        query as a convenience. Maybe we *should* have a parser per
        resource, it would make the regexps less hideous and painful.

    In the latter case:
     * *maybe* have a type, but *only* for the query.
     * how could we make it a little robust? could do
       more "producer / consumer" and have a little parser that can
       throw and error, or can check if it has collected what it should..

	Example of simple query:
	"min(last_15m):avg:stats.aws.vpc_prod_monitoring.ami.count{service_name:gnomes,aws-account-alias:vpc-production} > 3500"

	Example of multi alert query:
	"avg(last_15m):avg:system.disk.in_use{*} by {host,device} > 0.9"

    Outliers query (inherently a multi alert):
	"avg(last_1h):outliers(avg:system.fs.inodes.in_use{*} by {host},'dbscan',2) > 0"
*/

type subDatadogMonitor struct {
	// All
	Name    string
	Message string
	// end all
	// These could all be replaced with query
	TimeAggregate  string // check if not int
	TimeWindow     string
	SpaceAggregate string
	Metric         string
	Tags           []interface{}
	Keys           []interface{}
	Operator       string
	Threshold      string // correct type? this should be stored as for example warning.threshold 5
	// end of what could be replace if we only supported query
	// Outlier
	Algorithm string
	// end Outlier
	// Service Check
	Check string
	// end check
	// All
	ReNotifyInterval string // check Type
	NotifyNoData     bool
	NoDataTimeFrame  int
	// end all
}

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

func resourceDatadogQueryParser(d *schema.ResourceData, m *datadog.Monitor) (subDatadogMonitor, error) {

	monitor := subDatadogMonitor{}
	// Name
	if m.Name != d.Get("name") {
		log.Printf("[DEBUG] XX name was: %s found: %s", d.Get("name"), m.Name)
		monitor.Name = m.Name
	}

	// Message
	if m.Message != d.Get("message") {
		log.Printf("[DEBUG] XX message was: %s found: %s", d.Get("name"), m.Message)
		monitor.Message = m.Message
	}

	log.Printf("[DEBUG] storing notify_no_data: %v", m.Options.NotifyNoData)
	if m.Options.NotifyNoData != d.Get("notify_no_data") {
		log.Printf("[DEBUG] XX notify_no_data was: %t found: %t", d.Get("notify_no_data"), m.Options.NotifyNoData)
		monitor.NotifyNoData = m.Options.NotifyNoData
	}

	log.Printf("[DEBUG] storing nodata_time_frame: %v", m.Options.NoDataTimeframe)
	if m.Options.NoDataTimeframe != d.Get("no_data_timeframe") {
		log.Printf("[DEBUG] XX fy_no_data_timeframe was: %d found: %d", d.Get("no_data_timeframe"), m.Options.NoDataTimeframe)
		monitor.NoDataTimeFrame = m.Options.NoDataTimeframe
	}

	// TODO: decide which regexp testing m.Type, like this, or pull it
	// from a map
	if m.Type == "query alert" {
		log.Printf("[DEBUG] handling outlier (query) alert")
		/*
			See options for metric alert, with some added undocumented
				outlier options
		*/
	} else if m.Type == "metric alert" {
		log.Printf("[DEBUG] handling metric alert")
		/*
			time_aggr(time_window):space_aggr:metric{tags} [by {key}] operator #
			time_aggr avg, sum, max, min, change, or pct_change
			time_window last_#m (5, 10, 15, or 30), last_#h (1, 2, or 4), or last_1d
			space_aggr avg, sum, min, or max
			tags one or more tags (comma-separated), or *
			key a 'key' in key:value tag syntax; defines a separate alert for each tag in the group (multi-alert)
			operator <, <=, >, >=, ==, or !=
			# an integer or decimal number used to set the threshold
			If you are using the change or pct_change time aggregator, you can instead use change_aggr(time_aggr(time_window), timeshift):space_aggr:metric{tags} [by {key}] operator # with:
			change_aggr change, pct_change
			time_aggr avg, sum, max, min
			time_window last_#m (1, 5, 10, 15, or 30), last_#h (1, 2, or 4), or last_#d (1 or 2)
			timeshift #m_ago (5, 10, 15, or 30), #h_ago (1, 2, or 4), or 1d_ago
		*/
	} else if m.Type == "event alert" {
		log.Printf("[DEBUG] handling event alert")
		/*
			events('sources:nagios status:error,warning priority:normal tags: "string query"').rollup("count").last("1h")"
				event, the event query string:
					string_query free text query to match against event title and text.
				sources event sources (comma-separated).
				status event statuses (comma-separated). Valid options: error, warn, and info.
				priority event priorities (comma-separated). Valid options: low, normal, all.
				host event reporting host (comma-separated).
				tags event tags (comma-separated).
				excluded_tags exluded event tags (comma-separated).
				rollup the stats rollup method. count is the only supported method now.
				last the timeframe to roll up the counts. Examples: 60s, 4h. Supported timeframes: s, m, h and d.
		*/
	} else if m.Type == "service check" {
		log.Printf("[DEBUG] handling service check")
		/*
			"check".over(tags).last(count).count_by_status()
				check name of the check, e.g. datadog.agent.up
				tags one or more quoted tags (comma-separated), or "*". e.g.: .over("env:prod", "role:db")
				count must be at >= your max threshold (defined in the options). e.g. if you want to notify on 1 critical, 3 ok and 2 warn statuses count should be 3.
		*/
	}

	re := regexp.MustCompile("") // TODO unfuck me
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
					// This is tedious, use helper?
					if v, ok := d.GetOk("time_aggr"); ok {
						if n != v {
							log.Printf("[DEBUG] XX storing %s: %s", n1[i], n)
							monitor.TimeAggregate = n
						}
					}
				case n1[i] == "time_window": // Shared
					// This is tedious, use helper?
					if v, ok := d.GetOk("time_window"); ok {
						if n != v {
							log.Printf("[DEBUG] XX storing %s: %s", n1[i], n)
							monitor.TimeWindow = n
						}
					}
				case n1[i] == "space_aggr": // Shared
					// This is tedious, use helper?
					if v, ok := d.GetOk("space_aggr"); ok {
						if n != v {
							log.Printf("[DEBUG] XX storing %s: %s", n1[i], n)
							monitor.SpaceAggregate = n
						}
					}
				case n1[i] == "metric": // Shared
					if v, ok := d.GetOk("metric"); ok {
						if n != v {
							log.Printf("[DEBUG] XX storing %s. Old: %s, new: %s", n1[i], v, n)
							monitor.Metric = n
						}
					}
				case n1[i] == "tags": // Shared
					// TODO: move into helper function for tags and keys
					if v, ok := d.GetOk("tags"); ok {
						t := strings.Split(n, ",")
						temp := make([]interface{}, len(t))
						for i := range t {
							temp[i] = t[i]
						}
						log.Printf("[DEBUG] XX found: %s. Old: %v, new: %s", n1[i], v, temp)
						log.Printf("[DEBUG] XX found %s. Type old: %v, Type new: %s", n1[i], reflect.TypeOf(v), reflect.TypeOf(temp))

						if reflect.DeepEqual(v, temp) {
							log.Printf("[DEBUG] XX storing %s. Old: %v, new: %s", n1[i], v, temp)
							log.Printf("[DEBUG] XX storing %s. Type old: %v, Type new: %s", n1[i], reflect.TypeOf(v), reflect.TypeOf(temp))

							monitor.Tags = temp
						}
					}
				case n1[i] == "keys": // Shared
					// TODO: move into helper function for tags and keys
					if v, ok := d.GetOk("keys"); ok {
						t := strings.Split(n, ",")
						temp := make([]interface{}, len(t))
						for i := range t {
							temp[i] = t[i]
						}
						log.Printf("[DEBUG] XX found: %s. Old: %v, new: %s", n1[i], v, temp)
						log.Printf("[DEBUG] XX found %s. Type old: %v, Type new: %s", n1[i], reflect.TypeOf(v), reflect.TypeOf(temp))

						if reflect.DeepEqual(v, temp) {
							log.Printf("[DEBUG] XX storing %s. Old: %v, new: %s", n1[i], v, temp)
							log.Printf("[DEBUG] XX storing %s. Type old: %v, Type new: %s", n1[i], reflect.TypeOf(v), reflect.TypeOf(temp))

							monitor.Keys = temp
						}
					}
				case n1[i] == "operator": // Shared
					if v, ok := d.GetOk("operator"); ok {
						if n != v {
							log.Printf("[DEBUG] XX storing %s: %s", n1[i], n)
							monitor.Operator = n
						}
					}

				// TODO: this is different for resources that have
				//       warn/crit monitors (metric alerts) and others
				case n1[i] == "threshold": // Shared
					if v, ok := d.GetOk("treshold"); ok {
						if n != v {
							log.Printf("[DEBUG] XX storing %s: %s", n1[i], n)
							monitor.Threshold = n
						}
					}
				case n1[i] == "algorithm": // Outlier resource
					if v, ok := d.GetOk("algorithm"); ok {
						if n != v {
							log.Printf("[DEBUG] XX storing %s: %s", n1[i], n)
							monitor.Algorithm = n
						}
					}
				case n1[i] == "check": // Check resource
					if v, ok := d.GetOk("check"); ok {
						if n != v {
							log.Printf("[DEBUG] XX storing %s: %s", n1[i], n)
							monitor.Check = n
						}
					}
				case n1[i] == "renotify_interval": // Check resource
					if v, ok := d.GetOk("renotify_interval"); ok {
						if n != v {
							log.Printf("[DEBUG] XX storing %s: %s", n1[i], n)
							monitor.ReNotifyInterval = n
						}
					}
				}
			}
		}
	}

	return monitor, nil
}
