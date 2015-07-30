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
				Set: resourceDatadogGraphHash,
			},

			// TODO: support events.
		},
	}
}

func resourceDatadogGraphCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO: This should create graphs associated with dashboards.
	// it's a virtual resource, a la "resource_vpn_connection_route"
	// hence will need to do a bit of hacking to find out what dashboard.

	client := meta.(*datadog.Client)

	resourceDatadogGraphUpdate(d, meta)

	Id := int(time.Now().Unix())

	d.SetId(strconv.Itoa(Id)) // Use seconds since Epoch, needs to be a string when saving.

	log.Printf("[INFO] Dashboard ID: %s", Id)

	_, err := resourceDatadogGraphRetrieve(d.Get("dashboard_id").(int), client, d)

	if err != nil {
		return err
	}

	return nil
}

func resourceDatadogGraphRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	// TODO: Again, this conversion is annoying -the golang API returns and wants type Int, but Terraform uses String :|
	_, err := resourceDatadogGraphRetrieve(d.Get("dashboard_id").(int), client, d)

	if err != nil {
		return err
	}

	return nil
}

func resourceDatadogGraphRetrieve(id int, client *datadog.Client, d *schema.ResourceData) (*datadog.Graph, error) {
	// Here we go, we'll need to go into the dash and find ourselves
	dashboard, err := client.GetDashboard(id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving associated dashboard: %s", err)
	}

	for _, r := range dashboard.Graphs {
		// TODO: efficiently test if the are the same
		// for this POC we'll just match on title
		if r.Title == d.Get("title") {
			fmt.Println("Found matching title. Start setting/saving state.")
			d.Set("dashboard_id", d.Get("dashboard_id"))
			d.Set("title", r.Title)
			d.Set("description", r.Definition)
			d.Set("viz", r.Definition.Viz)
			// TODO: Add requests, a list of request IDs
			return &r, nil
		}
	}

	return nil, nil // TODO: should return something meaningful
}

func resourceDatadogGraphUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	dashboard, err := client.GetDashboard(d.Get("dashboard_id").(int))

	if err != nil {
		return err
	}

	for _, r := range dashboard.Graphs {
		// TODO: efficiently test if the are the same, use a Set and hashing?
		if r.Title == d.Get("title") {
			fmt.Println("Found matching title. Nothing to do here.")
			return nil
		}
	}

	log.Printf("[DEBUG] dashboard before added graph: %#v", dashboard)

	graph_definition := datadog.Graph{}.Definition

	graph_requests := datadog.Graph{}.Definition.Requests

	graph_definition.Viz = d.Get("viz").(string)

	log.Printf("[DEBUG] Checking if requests have changed.")

	if d.HasChange("request") {
		log.Printf("[DEBUG] Requests have changed.")
		o, n := d.GetChange("request")
		ors := o.(*schema.Set).Difference(n.(*schema.Set))
		nrs := n.(*schema.Set).Difference(o.(*schema.Set))

		// Now first loop through all the old routes and delete any obsolete ones
		for _, request := range ors.List() {
			m := request.(map[string]interface{})

			// Delete the route as it no longer exists in the config
			// TODO: implement
			// Delete the query as it no longer exists in the config
			log.Printf("[DEBUG] Deleting graph query %s", m["query"].(string))
			log.Printf("[DEBUG] Deleting graph stacked %s", m["stacked"].(bool))

			/*
			_, err := conn.DeleteRoute(&ec2.DeleteRouteInput{
				RouteTableID:         aws.String(d.Id()),
				DestinationCIDRBlock: aws.String(m["cidr_block"].(string)),
			})
			if err != nil {
				return err
			}
			*/
		}
		for _, request := range nrs.List() {
			m := request.(map[string]interface{})

			// Delete the route as it no longer exists in the config
			log.Printf("[DEBUG] Adding graph query %s", m["query"].(string))
			log.Printf("[DEBUG] Adding graph stacked %s", m["stacked"].(bool))
			graph_requests = append(graph_requests, GraphDefintionRequests{Query: m["query"].(string),
				Stacked: m["stacked"].(bool)})

			/*
			_, err := conn.DeleteRoute(&ec2.DeleteRouteInput{
				RouteTableID:         aws.String(d.Id()),
				DestinationCIDRBlock: aws.String(m["cidr_block"].(string)),
			})
			if err != nil {
				return err
			}
			*/
		}
	}

	/*
	for _, query := range requests {
		log.Printf("[DEBUG] query looks like: %#v", query)
		graph_requests = append(graph_requests,
		GraphDefintionRequests{Query: query,
						   Stacked: d.Get("stacked").(bool)})
	}
	*/

	graph_definition.Requests = graph_requests

	the_graph := datadog.Graph{Title: d.Get("title").(string), Definition: graph_definition}

	dashboard.Graphs = append(dashboard.Graphs, the_graph) // Should be done for each

	log.Printf("[DEBUG] dashboard after adding graph: %#v", dashboard)

	// Update/commit
	err = client.UpdateDashboard(dashboard)

	if err != nil {
		return err
	}

	return resourceDatadogGraphRead(d, meta)
}

func resourceDatadogGraphDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	// We'll need to find ourselves
	// * Get the dashboard(s) we are associated with. For this we'll need to access dashboard_id
	// * In the dashboard see if we are there or no (we'll use the title, for now)
	// * If not: return an error
	// * If yes: recreate the dashboard *without* our graph

	// We *could* use the hashing used by the AWS provider

	// Grab the dashboard and fetch it.
	dashboard, err := client.GetDashboard(d.Get("dashboard_id").(int))

	if err != nil {
		return fmt.Errorf("Error retrieving associated dashboard: %s", err)
	}

	// Now we will construct a new version of the graphs and call update with all the graphs
	// apart from the current one.
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

	// Can we do this?
	dashboard.Graphs = new_graphs

	// Call update with the Dashboard the new graph structure
	// TODO: there is a lot of overlap with the create function, let's create a helper function to build it
	// when we do a cleanup

	// Update/commit
	err = client.UpdateDashboard(dashboard)

	if err != nil {
		return err
	}

	_, err = resourceDatadogGraphRetrieve(d.Get("dashboard_id").(int), client, d)

	if err != nil {
		return err
	}

	return nil
}

func resourceDatadogGraphHash(v interface{}) int{
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
