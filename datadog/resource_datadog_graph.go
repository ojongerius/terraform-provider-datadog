package datadog

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/zorkian/go-datadog-api"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/hashcode"
)

// Work around the nested struct in https://github.com/zorkian/go-datadog-api/blob/master/dashboards.go#L16
type GraphDefintionRequests struct {
	Query   string `json:"q"`
	Stacked bool   `json:"stacked"`
}

func resourceDatadogGraph() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogGraphCreate,
		Read:   resourceDatadogGraphRead,
		Delete: resourceDatadogGraphDelete,
		Update: resourceDatadogGraphUpdate,
		//TODO: add Update

		Schema: map[string]*schema.Schema{
			"dashboard_id": &schema.Schema{
				Type:     schema.TypeInt,
				//Computed: true,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				//Computed: true, // TODO: what does this do?
				ForceNew: true,
				Optional: true,
			},
			"title": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"viz": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"request": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"query": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"stacked": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
						},
					},

				},
				Set: resourceDatadogRequestHash,
			},

			// TODO: support events.
		},
	}
}

func resourceDatadogGraphCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO: This should create graphs associated with dashboards.
	// it's a virtual resource, a la "resource_vpn_connection_route"
	// hence we will need to do a bit of hacking to find out what dashboard.

	resourceDatadogGraphUpdate(d, meta)

	Id := int(time.Now().Unix())

	d.SetId(strconv.Itoa(Id)) // Use seconds since Epoch, needs to be a string when saving.

	log.Printf("[INFO] Dashboard ID: %s", Id)

	err := resourceDatadogGraphRetrieve(d, meta)

	if err != nil {
		return err
	}

	return nil
}

func resourceDatadogGraphRead(d *schema.ResourceData, meta interface{}) error {
	err := resourceDatadogGraphRetrieve(d, meta)

	if err != nil {
		return err
	}

	return nil
}

func resourceDatadogGraphRetrieve(d *schema.ResourceData, meta interface{}) error {
	// Here we go, we'll need to go into the dash and find ourselves
	client := meta.(*datadog.Client)

	dashboard, err := client.GetDashboard(d.Get("dashboard_id").(int))

	if err != nil {
		return fmt.Errorf("Error retrieving associated dashboard: %s", err)
	}

	for _, g := range dashboard.Graphs {
		// TODO: efficiently test if the are the same. There are no ID, and there might be changes.
		// for now we'll use the title for as unique identifier. Interested in different strategies..

		if g.Title == d.Get("title") {
			fmt.Println("Found matching title. Start setting/saving state.")
			d.Set("dashboard_id", d.Get("dashboard_id"))
			d.Set("title", g.Title)
			d.Set("description", g.Definition)
			d.Set("viz", g.Definition.Viz)

			// Create an empty schema to hold all the requests.
			request := &schema.Set{F: resourceDatadogRequestHash}

			for _, r := range g.Definition.Requests {
				m := make(map[string]interface{})

				if r.Query != "" {
					m["query"] = r.Query
				}

				m["stacked"] = r.Stacked

				request.Add(m)
			}

			d.Set("request", request)

			return nil

		}
	}

	// If we are still around we've not found ourselves, set SetId to empty so Terraform will create us.
	d.SetId("")

	return nil
}

func resourceDatadogGraphUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	dashboard, err := client.GetDashboard(d.Get("dashboard_id").(int))

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] dashboard before added graph: %#v", dashboard)

	log.Printf("[DEBUG] Checking if requests have changed.")

	if d.HasChange("request") {
		graph_definition := datadog.Graph{}.Definition

		graph_requests := datadog.Graph{}.Definition.Requests

		graph_definition.Viz = d.Get("viz").(string)

		log.Printf("[DEBUG] Request has changed.")
		o, n := d.GetChange("request")
		ors := o.(*schema.Set).Difference(n.(*schema.Set))
		nrs := n.(*schema.Set).Difference(o.(*schema.Set))

		// Now first loop through all the old routes and delete any obsolete ones
		for _, request := range ors.List() {
			m := request.(map[string]interface{})

			// TODO: implement
			// Delete the query as it no longer exists in the config
			log.Printf("[DEBUG] Deleting graph query %s", m["query"].(string))
			log.Printf("[DEBUG] Deleting graph stacked %s", m["stacked"].(bool))

		}
		for _, request := range nrs.List() {
			m := request.(map[string]interface{})

			// Add the request
			log.Printf("[DEBUG] Adding graph query %s", m["query"].(string))
			log.Printf("[DEBUG] Adding graph stacked %s", m["stacked"].(bool))
			graph_requests = append(graph_requests, GraphDefintionRequests{Query: m["query"].(string),
				Stacked: m["stacked"].(bool)})
		}


		// TODO: should this not only by done when there is a change?
		graph_definition.Requests = graph_requests

		the_graph := datadog.Graph{Title: d.Get("title").(string), Definition: graph_definition}

		dashboard.Graphs = append(dashboard.Graphs, the_graph) // Should be done for each

		log.Printf("[DEBUG] dashboard after adding graph: %#v", dashboard)

		// Update/commit
		err = client.UpdateDashboard(dashboard)

		if err != nil {
			return err
		}
	} else {
		log.Printf("[DEBUG]No changes detected, nothing to do here.")
	}

	return nil
	//return resourceDatadogGraphRead(d, meta)
}

func resourceDatadogGraphDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	// Grab the dashboard and fetch it.
	dashboard, err := client.GetDashboard(d.Get("dashboard_id").(int))

	if err != nil {
		return fmt.Errorf("Error retrieving associated dashboard: %s", err)
	}

	// Now we will construct a new version of the graphs and call update with all the graphs
	// apart from the current one.
	// TODO: Use the set for this.
	new_graphs := []datadog.Graph{}
	for _, r := range dashboard.Graphs {
		// TODO: efficiently test if the are the same
		// for this POC we'll just match on title
		if r.Title == d.Get("title") {
			continue
		} else {
			// Keep this graph
			new_graphs = append(new_graphs, r)
		}
	}

	dashboard.Graphs = new_graphs

	// Update/commit
	err = client.UpdateDashboard(dashboard)

	if err != nil {
		return err
	}

	err = resourceDatadogGraphRetrieve(d, meta)

	if err != nil {
		return err
	}

	return nil
}

func resourceDatadogRequestHash(v interface{}) int{
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["query"];  ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	if v, ok := m["stacked"];  ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(bool)))
	}

	return hashcode.String(buf.String())
}
