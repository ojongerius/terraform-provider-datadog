package datadog

import (

	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

type GraphDefintionRequests struct {
        Query   string `json:"q"`
        Stacked bool   `json:"stacked"`
}

func resourceDatadogDashboard() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogDashboardCreate,
		Read:   resourceDatadogDashboardRead,
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

	// This should be handled by resource datadog_graph (used by timeseries and screenboards
	dash := datadog.Graph{}.Definition
	dash.Viz = "timeseries"

	r := datadog.Graph{}.Definition.Requests

	dash.Requests = append(r, GraphDefintionRequests{Query: "avg:system.mem.free{*}", Stacked: false})

	g := datadog.Graph{Title: "Graph title", Definition: dash}

	graphs := []datadog.Graph{}
	graphs = append(graphs, g) // Should be done for each

	log.Printf("[DEBUG] graphs now: %#v", graphs)

	opts.Graphs = graphs

	opts.Description = d.Get("description").(string)
	opts.Title = d.Get("title").(string)

	log.Printf("[DEBUG] Datadog create configuration: %#v", opts)

	dashboard , err := client.CreateDashboard(&opts)

	if err != nil {
		return err
	}

	d.SetId(strconv.Itoa(dashboard.Id))

	log.Printf("[INFO] Domain ID: %s", d.Id())

	IntId, err := strconv.Atoi(d.Id())

	if err != nil {
		return err
	}

	_, err = resourceDatadogDashboardRetrieve(IntId, client, d)

	if err != nil {
		return err
	}

	return nil
}

func resourceDatadogDashboardDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	log.Printf("[INFO] Deleting Dashboard: %s", d.Id())

	IntId, err := strconv.Atoi(d.Id())

	if err != nil {
		return err
	}

	// Destroy the domain
	err = client.DeleteDashboard(IntId)

	if err != nil {
		return fmt.Errorf("Error deleting Dashboard: %s", err)
	}

	return nil
}

func resourceDatadogDashboardRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	StringId, int_err := strconv.Atoi(d.Id())
	if int_err != nil {
		return int_err
	}

	_, err := resourceDatadogDashboardRetrieve(StringId, client, d)

	if err != nil {
		return err
	}

	return nil
}

func resourceDatadogDashboardRetrieve(id int, client *datadog.Client, d *schema.ResourceData) (**datadog.Dashboard, error) {
	resp, err := client.GetDashboard(id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving dashboard: %s", err)
	}

	d.Set("id", resp.Id)
	d.Set("descripton", resp.Description)
	d.Set("title", resp.Title)
	d.Set("graphs", resp.Graphs)

	return &resp, nil
}
