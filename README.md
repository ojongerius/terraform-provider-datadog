[![Build
status](https://travis-ci.org/ojongerius/terraform-provider-datadog.svg)](https://travis-ci.org/ojongerius/terraform-provider-datadog)

# terraform-provider-datadog

A [Terraform](https://github.com/hashicorp/terraform) plugin that provides resources for [Datadog](https://www.datadoghq.com/).

It currently supports 4 resources:

* *Service Checks*: datadog_service_check
* *Metric Alerts*: datadog_metric_alert (*experimental*)
* *Monitors*: datadog_monitor -originally contributed by [Vincenzo
  Prignano](https://github.com/vinceprignano) of [Segmentio](https://github.com/segmentio). This will be renamed to datadog_metric_alert in the future.
* *Timeboards*: datadog_dashboard
* *Graphs*: datadog_graph

Feel free to open new [issues](https://github.com/ojongerius/terraform-provider-datadog/issues) for extra resources or bugs you find. After finishing
polishing of the current resources I'm planning to add a
[Screenboard](https://github.com/ojongerius/terraform-provider-datadog/issues/4).

Want to contribute? Find a resource you want to add or work on an issue over
[here]( 
https://github.com/ojongerius/terraform-provider-datadog/issues).

##  Download
Download builds for Darwin, Linux and Windows from the [releases page](https://github.com/ojongerius/terraform-provider-datadog/releases/).

## Resources

### Service Checks

Example configuration:

``` HCL
resource "datadog_service_check" "bar" {
  name = "name for service check bar"
  message = "description for service check bar @pagerduty"
  check = "datadog.agent.up"
  check_count = 3
  tags = ["environment:foo", "host:bar"]

  notify_no_data = false
}
```

### Metric Alerts

Example configuration:

``` HCL
  name = "name for metric_alert foo"
  message = "description for metric_alert foo"

  metric = "aws.ec2.cpu"                 // Metric to monitor
  tags = ["environment:bar", "host:foo"] // List of tags to monitor
  keys = ["host"]                        // List of tag keys to alert on, enabling multi-alerts

  time_aggr = "avg"                      // avg, sum, max, min, change, or pct_change
  time_window = "last_1h"                // last_#m (5, 10, 15, 30), last_#h (1, 2, 4), or last_1d
  space_aggr = "avg"                     // avg, sum, min, or max
  operator = "<"                         // <, <=, >, >=, ==, or !=

  warning {                              // Creates alert with threshold 80
    threshold = 80
    notify = "@hipchat-<name>"
  }

  critical {                             // Creates alert with threshold 90
    threshold = 90
    notify = "@pagerduty"
  }

  notify_no_data = false                 // Do not alert on no data

}
```

### Monitors

Example configuration, _this resource will be renamed to datadog_metric_alert_:

``` HCL
resource "datadog_monitor" "baz" {
    name = "baz"
    message = "Description of monitor baz"

    metric = "aws.ec2.cpu"
    metric_tags = "*" // one or more comma separated tags (defaults to *)

    time_aggr = "avg" // avg, sum, max, min, change, or pct_change
    time_window = "last_1h" // last_#m (5, 10, 15, 30), last_#h (1, 2, 4), or last_1d
    space_aggr = "avg" // avg, sum, min, or max
    operator = "<" // <, <=, >, >=, ==, or !=

    warning {
        threshold = 0
        notify = "@hipchat-<name>"
    }

    critical {
        threshold = 0
        notify = "@pagerduty"
    }

    notify_no_data = false // Optional, defaults to true
}
```

### Dashboards

Example configuration:

``` HCL
resource "datadog_dashboard" "foo" {
    description = "description for dashboard foo"
    title = "title for dashboard foo bar"
    template_variable {
       name = "bar"
       prefix = "host"
       default = "host:bar.example.com"
    }
    template_variable {
       name = "baz"
       prefix = "host"
       default = "host:baz.example.com"
    }
}
```
### Graphs

Example configuration:

``` HCL
   resource "datadog_graph" "bar" {
       title = "Average Memory Free bar"
       dashboard_id = "${datadog_dashboard.foo.id}"
       description = "description for graph bar"
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
```

## Usage

Like any other Terraform interactions.

Tip: export `DATADOG_API_KEY` and `DATADOG_APP_KEY` as environment variables.

###Plan
```sh
> terraform plan
Refreshing Terraform state prior to plan...


The Terraform execution plan has been generated and is shown below.
Resources are shown in alphabetical order for quick scanning. Green resources
will be created (or destroyed and then created if an existing resource
exists), yellow resources are being changed in-place, and red resources
will be destroyed.

Note: You didn't specify an "-out" parameter to save this plan, so when
"apply" is called, Terraform can't guarantee this is what will execute.

+ datadog_dashboard.foo
    description: "" => "description for dashboard foo"
    title:       "" => "title for dashboard foo"

+ datadog_graph.bar
    dashboard_id:               "" => "0"
    description:                "" => "description for graph bar"
    request.#:                  "" => "3"
    request.1259113621.query:   "" => "avg:system.cpu.system{*}"
    request.1259113621.stacked: "" => "0"
    request.3179289285.query:   "" => "avg:system.cpu.user{*}"
    request.3179289285.stacked: "" => "0"
    request.458314230.query:    "" => "avg:system.mem.user{*}"
    request.458314230.stacked:  "" => "0"
    title:                      "" => "Average Memory Free bar"
    viz:                        "" => "timeseries"

+ datadog_monitor.baz
    critical.#:         "0" => "2"
    critical.notify:    "" => "@pagerduty"
    critical.threshold: "" => "0"
    message:            "" => "Description of monitor baz"
    metric:             "" => "aws.ec2.cpu"
    metric_tags:        "" => "*"
    name:               "" => "baz"
    notify_no_data:     "" => "1"
    operator:           "" => "<"
    space_aggr:         "" => "avg"
    time_aggr:          "" => "avg"
    time_window:        "" => "last_1h"
    warning.#:          "0" => "2"
    warning.notify:     "" => "@hipchat-<name>"
    warning.threshold:  "" => "0"


Plan: 3 to add, 0 to change, 0 to destroy.
```

###Apply

```sh
> terraform apply
datadog_dashboard.foo: Creating...
  description: "" => "description for dashboard foo"
  title:       "" => "title for dashboard foo"
datadog_monitor.baz: Creating...
  critical.#:         "0" => "2"
  critical.notify:    "" => "@pagerduty"
  critical.threshold: "" => "0"
  message:            "" => "Description of monitor baz"
  metric:             "" => "aws.ec2.cpu"
  metric_tags:        "" => "*"
  name:               "" => "baz"
  notify_no_data:     "" => "1"
  operator:           "" => "<"
  space_aggr:         "" => "avg"
  time_aggr:          "" => "avg"
  time_window:        "" => "last_1h"
  warning.#:          "0" => "2"
  warning.notify:     "" => "@hipchat-<name>"
  warning.threshold:  "" => "0"
datadog_monitor.baz: Creation complete
datadog_dashboard.foo: Creation complete
datadog_graph.bar: Creating...
  dashboard_id:               "" => "61249"
  description:                "" => "description for graph bar"
  request.#:                  "" => "3"
  request.1259113621.query:   "" => "avg:system.cpu.system{*}"
  request.1259113621.stacked: "" => "0"
  request.3179289285.query:   "" => "avg:system.cpu.user{*}"
  request.3179289285.stacked: "" => "0"
  request.458314230.query:    "" => "avg:system.mem.user{*}"
  request.458314230.stacked:  "" => "0"
  title:                      "" => "Average Memory Free bar"
  viz:                        "" => "timeseries"
datadog_graph.bar: Creation complete



Apply complete! Resources: 3 added, 0 changed, 0 destroyed.
```

## Development
### Running tests

#### Simple tests
```sh
> make test
go generate ./...
TF_ACC= go test ./...  -timeout=30s -parallel=4
?       github.com/ojongerius/terraform-provider-datadog[no test files]
ok      github.com/ojongerius/terraform-provider-datadog/datadog0.007s
go tool vet -asmdecl -atomic -bool -buildtags -copylocks -methods -nilfunc
-printf -rangeloops -shift -structtags -unsafeptr .
```

#### Acceptance tests

Much more extensive but need a valide Datadog API and APP key.

These tests will create and destroy real resources.

```sh
make testacc
```

### Building
Building defaults to the platform you run on, and depends on
[gox](https://github.com/mitchellh/gox). If you do not have it installed:

```sh
go get github.com/mitchellh/gox
```

```sh
> make bin
 go generate ./...
 Compiling for OS: darwin and ARCH: amd64
 Number of parallel builds: 8

 -->    darwin/amd64: github.com/ojongerius/terraform-provider-datadog
 Looking for Terraform install

 Moving terraform-provider-datadog_darwin_amd64 to
 /Applications/terraform/terraform-provider-datadog

 Resulting binary:

 -rwxr-xr-x 1 ojongerius staff 10442740 4 Aug 18:32
 /Applications/terraform/terraform-provider-datadog
```
