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
		Exists: resourceDatadogMonitorExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"check": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"count": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			// Metric and Monitor settings
			"metric": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"tags": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "*",
			},
			"time_aggr": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"time_window": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"space_aggr": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"operator": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"message": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			// Alert Settings
			"warning": &schema.Schema{
				Type:     schema.TypeMap,
				Optional:  true,
			},
			"critical": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
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
	// TODO: This is WIP, it might make more sense to have 2 separate resources.
	// A little duplication VS being able to handle if configs are optional/mandatory, etc

	log.Print("[DEBUG] building monitor struct")
	name := d.Get("name").(string)
	message := d.Get("message").(string)
	tags := d.Get("tags").(string)
	monitorType := d.Get("type").(string)
	var query string

	if d.Get("type").(string) == "service_check" {
		fmt.Println("It's a service check")
		// TODO: for now we'll let users pass this is as in, constructing is pretty fiddly and does not
		// add much value (AFAIC)
		query = d.Get("query").(string)
		// Example queries:
		//    'query': "zone_check".over("host:gnomes-i-4d82f8a4").last(3).count_by_status()'
		//    'query': '"ntp.in_sync".over("*").last(2).count_by_status()',
		//    'query': 'avg(last_5m):avg:aws.rds.cpuutilization{*} > 80',
		//    'query': 'avg(last_15m):avg:aws.elb.un_healthy_host_count{*} > 1',
		//	  'query': 'change(sum(last_1h),1h_ago):sum:puppet.failure.events{*} > 0',
		//    'query': '"datadog.agent.up".over("service_name:bulk-download").last(2).count_by_status()',
		//
		/*
		/query = fmt.Sprintf("\"%s\".over(\"%s)\").last(%s).count_by_status()", check,
			tags,
			count)
		*/
	} else {
		fmt.Println("It's a metric check")
		operator := d.Get("operator").(string)
		timeAggr := d.Get("time_aggr").(string)
		timeWindow := d.Get("time_window").(string)
		spaceAggr := d.Get("space_aggr").(string)
		metric := d.Get("metric").(string)
		query = fmt.Sprintf("%s(%s):%s:%s{%s} %s %s", timeAggr,
			timeWindow,
			spaceAggr,
			metric,
			tags,
			operator,
			d.Get(fmt.Sprintf("%s.threshold", typeStr)))
	}

	o := datadog.Options{
		NotifyNoData:    d.Get("notify_no_data").(bool),
		NoDataTimeframe: d.Get("no_data_timeframe").(int),
	}
	// TODO: handle notifications for service checks.

	m := datadog.Monitor{
		Type:    monitorType,
		Query:   query,
		Name:    fmt.Sprintf("[%s] %s", typeStr, name), // typeStr only for metrics
		Message: fmt.Sprintf("%s", message),
		Options: o,
	}

	return &m
}

// resourceDatadogMonitorCreate creates a monitor.
func resourceDatadogMonitorCreate(d *schema.ResourceData, meta interface{}) error {
	log.Print("[DEBUG] creating monitor")
	client := meta.(*datadog.Client)

	if d.Get("type").(string) == "service_check" {
		// TODO: remove "meh"
		log.Print("[DEBUG] Creating service check")
		m, err := client.CreateMonitor(buildMonitorStruct(d, "meh"))

		if err != nil {
			return fmt.Errorf("error creating service check: %s", err)
		}

		d.SetId(strconv.Itoa(m.Id))
		return nil
	}

	log.Print("[DEBUG] Creating metrics check")

	fmt.Println("XXXXX")
	fmt.Print(buildMonitorStruct(d, "warning"))

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
	log.Print("[DEBUG] deleting monitor")
	client := meta.(*datadog.Client)

	if d.Get("type").(string) == "service_check" {
		log.Print("[DEBUG] Deleting service check")
		ID, err := strconv.Atoi(d.Id())
		if err != nil {
			return err
		}

		err = client.DeleteMonitor(ID)
		if err != nil {
			return err
		}
		return nil
	}

	log.Print("[DEBUG] Deleting metrics check")
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

// resourceDatadogMonitorExists verifies a monitor exists.
func resourceDatadogMonitorExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	// Exists - This is called to verify a resource still exists. It is called prior to Read,
	// and lowers the burden of Read to be able to assume the resource exists.

	log.Print("[DEBUG] verifying monitor exists")
	client := meta.(*datadog.Client)

	if d.Get("type").(string) == "service_check" {
		log.Print("[DEBUG] Deleting service check")
		ID, err := strconv.Atoi(d.Id())
		if err != nil {
			return false, err
		}
		_, err = client.GetMonitor(ID)

		if strings.EqualFold(err.Error(), "API error: 404 Not Found") {
			log.Printf("[DEBUG] monitor does not exist: %s", err)
			return false, err
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	log.Print("[DEBUG] Deleting metrics check")
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

	if exists == false {
		return false, nil
	}

	return true, nil
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

	client := meta.(*datadog.Client)

	if d.Get("type").(string) == "service_check" {
		body := buildMonitorStruct(d, "meh")

		ID, err := strconv.Atoi(d.Id())
		if err != nil {
			return err
		}

		body.Id = ID
		err = client.UpdateMonitor(body)

		if err != nil {
			return fmt.Errorf("error updating warning: %s", err.Error())
		}
		return nil
	}

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
