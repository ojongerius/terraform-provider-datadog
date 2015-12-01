package datadog

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
	"log"
	"strconv"
	"strings"
)

func resourceDatadogGenericRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*datadog.Client)
	for _, v := range strings.Split(d.Id(), "__") {
		if v == "" {
			return fmt.Errorf("Id not set.")
		}
		ID, iErr := strconv.Atoi(v)

		if iErr != nil {
			return iErr
		}

		m, err := client.GetMonitor(ID)

		if err != nil {
			return err
		}

		err = resourceDatadogQueryParser(d, m)

		if err != nil {
			return err
		}
	}

	return nil
}

func resourceDatadogGenericExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	// Exists - This is called to verify a resource still exists. It is called prior to Read,
	// and lowers the burden of Read to be able to assume the resource exists.
	//
	// If the resource is no longer present in remote state, calling SetId with an empty string will
	// signal its removal.

	// TODO: this can be the same for each resource

	client := meta.(*datadog.Client)

	// Set default to false
	exists := false
	expectedCount := 0
	actualCount := 0
	for _, v := range strings.Split(d.Id(), "__") {
		expectedCount += 1
		if v == "" {
			log.Printf("[DEBUG] Could not parse IDs: %s", v)
			return false, fmt.Errorf("Id not set.")
		}
		ID, iErr := strconv.Atoi(v)

		if iErr != nil {
			log.Printf("[DEBUG] Received error converting string: %s", iErr)
			return false, iErr
		}
		_, err := client.GetMonitor(ID)
		if err != nil {

			if strings.EqualFold(err.Error(), "API error 404 Not Found: {\"errors\":[\"Monitor not found\"]}") {
				log.Printf("[DEBUG] monitor %s does not exist: %s", v, err)
				exists = false
				continue
			} else {
				log.Printf("[DEBUG] received error getting monitor %s: %s", v, err)
				e = err
				continue
			}
		}
		actualCount += 1
		log.Printf("[DEBUG] found monitor %s", v)
		exists = true
	}

	if expectedCount != actualCount && actualCount > 0 {
		return false, fmt.Errorf("found %d monitors, but expected %d. Please taint this resource with 'terraform taint $RESOURCETYPE.$RESOURCENAME' and recreate it with 'terraform apply'", actualCount, expectedCount)
	}

	return exists, e
}
