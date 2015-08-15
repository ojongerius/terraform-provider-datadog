package datadog

import (
	"github.com/zorkian/go-datadog-api"
)

func createPlaceholderGraph() []datadog.Graph {
	// Return a dummy placeholder graph.
	// This should be used when creating new dashboards, or removing the last
	// in a board.
	// Background; An API call to create or update dashboards (Timeboards) will
	// fail if it contains zero graphs. This is a bug in the Datadog API,
	// as dashboards *can* exist without any graphs.

	graph_definition := datadog.Graph{}.Definition
	graph_definition.Viz = "timeseries"
	r := datadog.Graph{}.Definition.Requests
	graph_definition.Requests = append(r, GraphDefintionRequests{Query: "avg:system.mem.free{*}", Stacked: false})
	graph := datadog.Graph{Title: "Mandatory placeholder graph", Definition: graph_definition}
	graphs := []datadog.Graph{}
	graphs = append(graphs, graph)
	return graphs
}
