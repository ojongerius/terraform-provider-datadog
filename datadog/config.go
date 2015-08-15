package datadog

import (
	"log"

	"github.com/zorkian/go-datadog-api"
)

type Config struct {
	api_key string
	app_key string
}

// Client() returns a new client for accessing datadog.
//
func (c *Config) Client() (*datadog.Client, error) {

	client := datadog.NewClient(c.api_key, c.app_key)

	log.Printf("[INFO] Datadog Client configured ")

	return client, nil
}
