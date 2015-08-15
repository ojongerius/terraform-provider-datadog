package datadog

import (
	"github.com/zorkian/go-datadog-api"
)

func createPlaceholderGraph() []datadog.Graph {
	// Return a dummy placeholder graph -An API call to create or update a dashboard will
	// fail if there are no graphs

	graph_definition := datadog.Graph{}.Definition
	graph_definition.Viz = "timeseries"
	r := datadog.Graph{}.Definition.Requests
	graph_definition.Requests = append(r, GraphDefintionRequests{Query: "avg:system.mem.free{*}", Stacked: false})
	graph := datadog.Graph{Title: "Mandatory placeholder graph", Definition: graph_definition}
	graphs := []datadog.Graph{}
	graphs = append(graphs, graph) // Should be done for each
	return graphs
}
