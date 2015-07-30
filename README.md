# terraform-provider-datadog

Warning: This plugin is work in progress.

A terraform plugin that provides resources for Datadog.

# Build

```
./build.sh

Compiling for OS: darwin and ARCH: amd64
Number of parallel builds: 8

-->    darwin/amd64: github.com/ojongerius/terraform-provider-datadog
Looking for Terraform install

Moving terraform-provider-datadog_darwin_amd64 to /Applications/terraform/terraform-provider-datadog

Resulting binary:

-rwxr-xr-x 1 ojongerius staff 10442740 30 Jul 18:19 /Applications/terraform/terraform-provider-datadog
```

# Example config

```
variable "api_key" { default = "" }
variable "app_key" { default = "" }

resource "datadog_dashboard" "foo" {
    description = "baz"
    title = "bar"
}

resource "datadog_graph" "baz" {
    title = "Average Memory Free baz"
    dashboard_id = "${datadog_dashboard.foo.id}"
    description = "baz"
    title = "bar"
    viz =  "timeseries"
    request {
        query =  "avg:system.cpu.system{*}"
        stacked = false
    }
    request {
        query =  "avg:system.cpu.user{*}"
        stacked = false
    }
    request {
        query =  "avg:system.mem.user{*}"
        stacked = false
    }

}

resource "datadog_monitor" test_monitor {
  name = "foo"
  message = "Something that describes this monitor"

  metric = "aws.ec2.cpu"
  metric_tags = "*" // one or more comma separated tags (defaults to *)

  time_aggr = "avg" // avg, sum, max, min, change, or pct_change
  time_window = "last_1d" // last_#m (5, 10, 15, 30), last_#h (1, 2, 4), or last_1d
  space_aggr = "avg" // avg, sum, min, or max
  operator = "<" // <, <=, >, >=, ==, or !=

  warning {
    threshold = 0
    notify = "@slack-<name>"
  }

  critical {
    threshold = 0
    notify = "@pagerduty"
  }

  notify_no_data = false // Optional, defaults to true
}
```
