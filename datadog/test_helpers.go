package datadog

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/zorkian/go-datadog-api"
)

func destroyHelper(s *terraform.State, client *datadog.Client) error {
	for _, r := range s.RootModule().Resources {
		i, _ := strconv.Atoi(r.Primary.ID)
		if _, err := client.GetMonitor(i); err != nil {
			if strings.Contains(err.Error(), "404 Not Found") {
				continue
			} else {
				return fmt.Errorf("Received an error retreieving monitor %s", err)
			}
		} else {
			return fmt.Errorf("Monitor still exists. %s", err)
		}
	}
	return nil
}

func existsHelper(s *terraform.State, client *datadog.Client) error {
	for _, r := range s.RootModule().Resources {
		i, _ := strconv.Atoi(r.Primary.ID)
		if _, err := client.GetMonitor(i); err != nil {
			return fmt.Errorf("Received an error retrieving monitor %s", err)
		}
	}
	return nil
}
