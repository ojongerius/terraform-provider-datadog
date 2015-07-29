package datadog

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

func resourceDatadogGraph() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogGraphCreate,
		Read:   resourceDatadogGraphRead,
		Delete: resourceDatadogGraphDelete,
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
			// TODO: support events.
		},
	}
}

func resourceDatadogGraphCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO: This should create graphs associated with dashboards.
	// it's a virtual resource, a la "resource_vpn_connection_route"
	// hence will need to do a bit of hacking to find out what dashboard.

	client := meta.(*datadog.Client)

	// To create a dashboard we'll have to:
	// * if it does not exist: what should we do here? The dashboard
	//   resource should handle this AFAICS
	// * if no associated dashboard exist: remove it/taint/remove from state
	// * if the dashboard does exist, and the graph is there, NOOP.
	// * if the dashboard does exist, and the graph is not there, create.
	// * if the dashboard does exist, and the graph is different; we
	//   can't see this, as graphs have no IDs in DD. It will be a delete
	//   and create event.

	// Here: the config has a (dynamically filled) dashboard ID, so
	// we'll read the dashboard and see if:
	// * diff the existing graph to see if it needs updating
	//  OR
	// * Create the graph

	//DashId, conv_err := strconv.Atoi(d.Get("dashboard_id"))
	//DashId, conv_err := strconv.Atoi(d.Get("dashboard_id"))

	//if conv_err != nil {
		//return conv_err
	//}

	// Shall we just get it ourselves?
	//dashboard, err := resourceDatadogDashboardRetrieve(DashId, client, d)
	/*
	Trying to workaround:

	--> darwin/amd64 error: exit status 2
    	Stderr: # github.com/hashicorp/terraform/builtin/providers/datadog
    	../terraform/builtin/providers/datadog/resource_datadog_graph.go:117: dashboard.Graphs undefined (type **datadog.Dashboard has no field or method Graphs)
	 */
	dashboard, err := client.GetDashboard(d.Get("dashboard_id").(int))

	if err != nil {
		return err
	}

	// Look in dashboard and see if our graph it is in there?
	// This is fun; graphs do not have an ID, so it has be to a 100% match :(
	// if we made it this far, we'll have to create the Graph, which means
	// we have the privilege of updating the dashboard.

	for _, r := range dashboard.Graphs {
		// TODO: efficiently test if the are the same, can use a Set and hashing.
		// for this POC we'll just match on title
		// If it is there, but different, (re)create it.
		if r.Title == d.Get("title") {
			fmt.Println("Found matching title. Nothing to do here.")
			return nil
		}
	}

	// If we made it this far, we are going to:
	// * Create the graph object
	// * Update the dashboard with the graph

	log.Printf("[DEBUG] dashboard before added graph: %#v", dashboard)

	graph_definition := datadog.Graph{}.Definition


	// Just create an empty struct and let the request resources handle creation of requests.
	graph_requests := datadog.Graph{}.Definition.Requests

	graph_definition.Viz = d.Get("viz").(string)
	graph_definition.Requests = graph_requests
	the_graph := datadog.Graph{Title: d.Get("title").(string), Definition: graph_definition}
	dashboard.Graphs = append(dashboard.Graphs, the_graph) // Should be done for each

	log.Printf("[DEBUG] dashboard after adding graph: %#v", dashboard)

	// Update/commit
	err = client.UpdateDashboard(dashboard)

	if err != nil {
		return err
	}

	Id := int(time.Now().Unix())

	d.SetId(strconv.Itoa(Id)) // Use seconds since Epoch, needs to be a string when saving.

	log.Printf("[INFO] Dashboard ID: %s", Id)

	_, err = resourceDatadogGraphRetrieve(d.Get("dashboard_id").(int), client, d)

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
