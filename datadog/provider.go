package datadog

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DATADOG_API_KEY", nil), // TODO: not fetched from env?
			},
			"app_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DATADOG_APP_KEY", nil),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			// TODO this is where the other resources will be mapped
			"datadog_dashboard": resourceDatadogDashboard(),
			"datadog_graph": resourceDatadogGraph(),
			"datadog_monitor": resourceDatadogMonitor(),
		},

		ConfigureFunc: providerConfigure,
	}
}

// TODO suck because client of lib only returns a client, no error
func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		api_key: d.Get("api_key").(string),
		app_key: d.Get("app_key").(string),
	}

	log.Println("[INFO] Initializing Datadog client")
	return config.Client()
}
