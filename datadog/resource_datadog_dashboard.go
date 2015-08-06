package datadog

import (

	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

func resourceDatadogDashboard() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogDashboardCreate,
		Read:   resourceDatadogDashboardRead,
		Exists: resourceDatadogDashboardExists,
		Delete: resourceDatadogDashboardDelete,
		// TODO: add Update

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
				//Required: true,
				//ForceNew: true,
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
		},
	}
}

func resourceDatadogDashboardCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	opts := datadog.Dashboard{}
	opts.Description = d.Get("description").(string)
	opts.Title = d.Get("title").(string)

	// A dummy placeholder graph (a graph is mandatory)
	// We cannot get rid of this placeholder graph, so we'll nuke it when applying graphs
	graph_definition := datadog.Graph{}.Definition
	graph_definition.Viz = "timeseries"
	r := datadog.Graph{}.Definition.Requests
    graph_definition.Requests = append(r, GraphDefintionRequests{Query: "avg:system.mem.free{*}", Stacked: false})
    graph := datadog.Graph{Title: "Mandatory default graph title", Definition: graph_definition}
    graphs := []datadog.Graph{}
    graphs = append(graphs, graph) // Should be done for each
    opts.Graphs = graphs

	dashboard , err := client.CreateDashboard(&opts)

	if err != nil {
		return fmt.Errorf("Error creating Dashboard: %s", err)
	}

	d.SetId(strconv.Itoa(dashboard.Id))

	err = resourceDatadogDashboardRead(d, meta)

	if err != nil {
		return fmt.Errorf("Error retrieving Dashboard: %s", err)
	}

	return nil
}

func resourceDatadogDashboardDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	log.Printf("[DEBUG] Deleting Dashboard: %s", d.Id())

	id, _ := strconv.Atoi(d.Id())

	err := client.DeleteDashboard(id)

	if err != nil {
		return fmt.Errorf("Error deleting Dashboard: %s", err)
	}

	return nil
}

func resourceDatadogDashboardExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*datadog.Client)

	id, _ := strconv.Atoi(d.Id())

	_, err := client.GetDashboard(id)

	if err != nil {
		if strings.EqualFold(err.Error(), "API error: 404 Not Found") {
			return false, nil
		}

		return false, fmt.Errorf("Error retrieving dashboard: %s", err)
	}

	return true, nil
}

func resourceDatadogDashboardRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	id, _ := strconv.Atoi(d.Id())

	resp, err := client.GetDashboard(id)

	if err != nil {
		return fmt.Errorf("Error retrieving dashboard: %s", err)
	}

	d.Set("id", resp.Id)
	d.Set("descripton", resp.Description)
	d.Set("title", resp.Title)
	d.Set("graphs", resp.Graphs)

	return nil
}
