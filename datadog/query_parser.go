package datadog

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
	"log"
	"reflect"
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

type subDatadogMonitor struct {
	Name             string
	Message          string
	Notify           string // how would we store this? .Set(fmt.Sprintf("%s.notify", level), v)
	TimeAggregate    string // check if not int
	TimeWindow       string
	SpaceAggregate   string
	Metric           string
	Tags             []interface{}
	Keys             []interface{}
	Operator         string
	Threshold        string // correct type? this should be stored as for example warning.threshold 5
	Algorithm        string
	Check            string
	ReNotifyInterval string // check Type
	NotifyNoData     bool
	NoDataTimeFrame  int
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
	re := regexp.MustCompile(`\[([a-zA-Z]+)\]\s(.+)`)
	// Find check name
	r := re.FindStringSubmatch(m.Name)
	if r == nil {
		return monitor, fmt.Errorf("Name parser error: string match returned nil")
	}
	if len(r) < 3 {
		return monitor, fmt.Errorf("Name parser error. Expected: 3. Got: %d", len(r))
	}
	level := r[1] // Store this so we can save the contact for in the right place (see below)
	log.Printf("[DEBUG] found level %s", level)
	log.Printf("[DEBUG] found name %s", r[2])

	if r[2] != d.Get("name") {
		log.Printf("[DEBUG] XX name was: %s found: %s", d.Get("name"), r[2])
		monitor.Name = r[2]
	}

	// Message
	res := strings.Split(m.Message, " @")
	if res == nil {
		return monitor, fmt.Errorf("Message parser error: string split returned nil")
	}

	log.Printf("[DEBUG] found message %s", res[0])
	if res[0] != d.Get("message") {
		log.Printf("[DEBUG] XX message was: %s found: %s", d.Get("name"), res[0])
		monitor.Message = res[0]
	}

	levelMap := d.Get(level).(map[string]interface{})

	for k, v := range res {
		if k == 0 {
			// The message is the first element, move on to the contact
			// TODO: handle cases where at-mentions are embedded/nested *in* the messages.
			continue
		}
		log.Printf("[DEBUG] found %s.notify: %s", level, levelMap["notify"])
		// TODO: this will
		if fmt.Sprintf("@%s", v) != levelMap["notify"] {
			log.Printf("[DEBUG] XX %s.notify was: %s found: %s", level, levelMap["notify"], level, v)
			monitor.Notify = v
		}
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

	return monitor, nil
}
