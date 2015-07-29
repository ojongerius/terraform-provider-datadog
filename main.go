package main

import (
    "github.com/hashicorp/terraform/plugin"

    "github.com/ojongerius/terraform-provider-datadog/datadog"
)

func main() {
    plugin.Serve(&plugin.ServeOpts{
        ProviderFunc: datadog.Provider,
    })
}
