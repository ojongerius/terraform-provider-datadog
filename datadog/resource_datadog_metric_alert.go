package datadog

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/zorkian/go-datadog-api"

	"github.com/hashicorp/terraform/helper/schema"
)

// resourceDatadogMetricAlert is a Datadog monitor resource
func resourceDatadogMetricAlert() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogMetricAlertCreate,
		Read:   resourceDatadogMetricAlertRead,
		Update: resourceDatadogMetricAlertUpdate,
		Delete: resourceDatadogMetricAlertDelete,
		Exists: resourceDatadogMetricAlertExists,

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

// resourceDatadogMetricAlertCreate creates a monitor.
func resourceDatadogMetricAlertCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	w, err := client.CreateMonitor(buildMetricAlertStruct(d, "warning"))

	if err != nil {
		return fmt.Errorf("error creating warning: %s", err)
	}

	c, cErr := client.CreateMonitor(buildMetricAlertStruct(d, "critical"))

	if cErr != nil {
		return fmt.Errorf("error creating warning: %s", cErr)
	}

	log.Printf("[DEBUG] Saving IDs: %s__%s", strconv.Itoa(w.Id), strconv.Itoa(c.Id))

	d.SetId(fmt.Sprintf("%s__%s", strconv.Itoa(w.Id), strconv.Itoa(c.Id)))

	return nil
}

// resourceDatadogMetricAlertDelete deletes a monitor.
func resourceDatadogMetricAlertDelete(d *schema.ResourceData, meta interface{}) error {
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

// resourceDatadogMetricAlertExists verifies a monitor exists.
func resourceDatadogMetricAlertExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	// Exists - This is called to verify a resource still exists. It is called prior to Read,
	// and lowers the burden of Read to be able to assume the resource exists.

	client := meta.(*datadog.Client)

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
			if strings.EqualFold(err.Error(), "API error: 404 Not Found") {
				log.Printf("[DEBUG] monitor does not exist: %s", err)
				exists = false
				continue
			} else {
				e = err
				continue
			}
		}
		exists = true
	}

	return exists, nil
}

// resourceDatadogMetricAlertRead synchronises Datadog and local state .
func resourceDatadogMetricAlertRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*datadog.Client)
	for _, v := range strings.Split(d.Id(), "__") {
		if v == "" {
			return fmt.Errorf("Id not set.")
		}
		ID, iErr := strconv.Atoi(v)

		if iErr != nil {
			return iErr
		}

		m, err := client.GetMonitor(ID)

		if err != nil {
			return err
		}

		err = resourceDatadogQueryParser(d, m, "metric_alert")

		if err != nil {
			return err
		}
	}

	return nil
}

// resourceDatadogQueryParser takes d, with resource data, m containing a monitoring and resourceType a string with the resource name/type.
func resourceDatadogQueryParser(d *schema.ResourceData, m *datadog.Monitor, resourceType string) error {

	/*
		This is the first iteration of of a generic query parser. Should it be renamed

	*/
	// Name -this is identical across resources.
	re := regexp.MustCompile(`\[([a-zA-Z]+)\]\s(.+)`)
	r := re.FindStringSubmatch(m.Name) // TODO: test if something is in fact there
	level := r[1]                      // Store this so we can save the contact for in the right place (see below)
	log.Printf("[DEBUG] found level %s", level)
	log.Printf("[DEBUG] storing %s", r[2])
	d.Set("name", r[2])

	// Message -this would be identical across resources too
	res := strings.Split(m.Message, " @")            // TODO: use must compile for this one
	log.Printf("[DEBUG] storing message %s", res[0]) // TODO: make robust
	d.Set("message", res[0])
	for k, v := range res {
		if k == 0 {
			// The message is the first element, move on to the contacts TODO: handle cases where at-mentions
			// are embeded/nested *in* the messages.
			continue
		}
		log.Printf("[DEBUG] storing %s.notify: %s", level, v)
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
		// TODO: this depends on the resource at hand, one generic switch should catch all
		r2 := subMatches[k]
		for i, n := range r2 {
			if n != "" {
				switch {
				case n1[i] == "time_aggr":
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("time_aggr", n)
				case n1[i] == "time_window":
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("time_window", n)
				case n1[i] == "space_aggr":
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("space_aggr", n)
				case n1[i] == "metric":
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("metric", n)
				case n1[i] == "tags":
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("tags", n)
				case n1[i] == "keys":
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set("keys", n)
				case n1[i] == "operator":
					d.Set("operator", n)
				case n1[i] == "threshold":
					log.Printf("[DEBUG] storing  %s", n1[i])
					d.Set(fmt.Sprintf("%s.threshold", level), n)
				}
			}
		}

	}
	log.Printf("[DEBUG] storing  %v", m.Options.NotifyNoData)
	d.Set("notify_no_data", m.Options.NotifyNoData) // TODO: Need to convert/assert bool?
	log.Printf("[DEBUG] storing  %v", m.Options.NoDataTimeframe)
	d.Set("no_data_timeframe", m.Options.NoDataTimeframe) // TODO: Need to convert/assert int?

	return nil
}

// resourceDatadogMetricAlertUpdate updates a monitor.
func resourceDatadogMetricAlertUpdate(d *schema.ResourceData, meta interface{}) error {
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

	warningBody := buildMetricAlertStruct(d, "warning")
	criticalBody := buildMetricAlertStruct(d, "critical")

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
