package datadog

import (
	"log"

    "github.com/zorkian/go-datadog-api"
)

type Config struct {
	APIKey string
	APPKey string
}

// Client() returns a new client for accessing datadog.
//
func (c *Config) Client() (*datadog.Client, error) {

	// TODO: NewClient does not return err, if the library sucks we'll create our own.
	//client, err := datadog.NewClient(c.APIKey, c.APPKey)
	client := datadog.NewClient(c.APIKey, c.APPKey)

	log.Printf("[INFO] Datadog Client configured ")

	return client, nil
}
