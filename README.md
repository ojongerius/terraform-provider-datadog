[![Build
status](https://travis-ci.org/ojongerius/terraform-provider-datadog.svg)](https://travis-ci.org/ojongerius/terraform-provider-datadog)

# terraform-provider-datadog

A [Terraform](https://github.com/hashicorp/terraform) plugin that provides resources for [Datadog](https://www.datadoghq.com/).

It currently supports 3 resources based on the Datadog monitor originally contributed by [Vincenzo Prignano](https://github.com/vinceprignano) of [Segmentio](https://github.com/segmentio).

* *Service Checks*: datadog_service_check
* *Metric Alerts*: datadog_metric_alert
* *Outlier Alerts*: datadog_outlier_alert, see [introducing-outlier-detection-in-datadog](https://www.datadoghq.com/blog/introducing-outlier-detection-in-datadog/).

Feel free to open new [issues](https://github.com/ojongerius/terraform-provider-datadog/issues) for extra resources or bugs you find.

Want to contribute? Find a resource you want to add or work on an issue over
[here]( 
https://github.com/ojongerius/terraform-provider-datadog/issues).

##  Download
Download builds for Darwin, Linux and Windows from the [releases page](https://github.com/ojongerius/terraform-provider-datadog/releases/). Pre-release is rebuild on each merge to master.

## Resources
### Service Checks

This plugin will create a monitor, but not a service check. By default it will
monitor reports from all hosts that run a given service check.

The flow would be:

* Run service checks on host(s)
* Create monitors for those service checks using this plugin

Filter which hosts / groups are monitored by using [tags](http://docs.datadoghq.com/guides/tagging/).

Example configuration:

``` HCL
resource "datadog_service_check" "foo" {
    name = "name for service check foo"
    message           = <<EOF
{{#is_alert}}Service check foo is critical{/is_alert}}
{{#is_warning}}Service check foo is at warning level{{/is_warning}}
{{#is_recovery}}Service check foo has recovered{{/is_recovery}}
Notify: @hipchat-channel
EOF
    check = "datadog.agent.up"
    check_count = 3
    tags = ["environment:foo", "host:bar"]
    keys = ["foo", "bar"]
    thresholds {
        ok       = 1 // Optional
        warning  = 2 // Optional
        critical = 3 // Required, formally known as "threshold"
    }

    notify_no_data = false
}
```

### Metric Alerts

Example configuration:

``` HCL
resource "datadog_metric_alert" "bar" {
    name        = "TF: bar"
    message           = <<EOF
{{#is_alert}}Metric alert check bar is critical{/is_alert}}
{{#is_warning}}Metric alert bar is warning{{/is_warning}}
{{#is_recovery}}Metric alert bar has recovered{{/is_recovery}}
Notify: @hipchat-channel
EOF
    metric      = "datadog.dogstatsd.packet.count"
    tags        = ["*"]
    keys        = ["host"]
    time_aggr   = "avg"
    time_window = "last_1h"
    operator    = ">"
    notify_no_data = 0
    space_aggr  = "sum"
    thresholds {
        ok       = 1 // Optional
        warning  = 2 // Optional
        critical = 3 // Required, formally known as "threshold"
    }
}
```

### Outlier Alerts

Example configuration:

``` HCL
resource "datadog_outlier_alert" "foo" {
  name = "name for outlier_alert foo"
  message = "description for outlier_alert @hipchat-channel"

  algorithm = "mad"

  metric = "system.load.5"
  tags = ["environment:foo", "host:foo"]
  keys = ["host"]

  time_aggr = "avg"       // avg, sum, max, min, change, or pct_change
  time_window = "last_1h" // last_#m (5, 10, 15, 30), last_#h (1, 2, 4), or last_1d
  space_aggr = "avg"      // avg, sum, min, or max

  threshold = 3.0         // tolerance

  notify_no_data = false

}
```

## Usage

Like any other Terraform interactions.

Tip: export `DATADOG_API_KEY` and `DATADOG_APP_KEY` as environment variables.

###Plan
```sh
ojongerius@hipster  ~/dev/go/datadog  terraform plan
Refreshing Terraform state prior to plan...


The Terraform execution plan has been generated and is shown below.
Resources are shown in alphabetical order for quick scanning. Green resources
will be created (or destroyed and then created if an existing resource
exists), yellow resources are being changed in-place, and red resources
will be destroyed.

Note: You didn't specify an "-out" parameter to save this plan, so when
"apply" is called, Terraform can't guarantee this is what will execute.

+ datadog_metric_alert.statsd_packet_count
    keys.#:            "0" => "1"
    keys.0:            "" => "host"
    message:           "" => "statsd packet count {{host.hostname}}"
    metric:            "" => "datadog.dogstatsd.packet.count"
    name:              "" => "TF: stats_packet_count"
    notify:            "" => "@ojongerius@warning.com"
    notify_no_data:    "" => "0"
    operator:          "" => ">"
    renotify_interval: "" => "0"
    space_aggr:        "" => "sum"
    tags.#:            "0" => "1"
    tags.0:            "" => "*"
    threshold:         "" => "2"
    time_aggr:         "" => "avg"
    time_window:       "" => "last_1h"


Plan: 1 to add, 0 to change, 0 to destroy.
```

###Apply

```sh
ojongerius@hipster  ~/dev/go/datadog  terraform apply
datadog_metric_alert.statsd_packet_count: Creating...
  keys.#:            "0" => "1"
  keys.0:            "" => "host"
  message:           "" => "statsd packet count {{host.hostname}}"
  metric:            "" => "datadog.dogstatsd.packet.count"
  name:              "" => "TF: stats_packet_count"
  notify:            "" => "@ojongerius@warning.com"
  notify_no_data:    "" => "0"
  operator:          "" => ">"
  renotify_interval: "" => "0"
  space_aggr:        "" => "sum"
  tags.#:            "0" => "1"
  tags.0:            "" => "*"
  threshold:         "" => "2"
  time_aggr:         "" => "avg"
  time_window:       "" => "last_1h"
datadog_metric_alert.statsd_packet_count: Creation complete

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.

The state of your infrastructure has been saved to the path
below. This state is required to modify and destroy your
infrastructure, so keep it safe. To inspect the complete state
use the `terraform show` command.

State path: terraform.tfstate
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

Much more extensive but need a valid Datadog API and APP key.

These tests will both create and destroy real resources.

```sh
ojongerius@hipster  ~/gocode/src/github.com/ojongerius/terraform-provider-datadog   KISS ●  make
testacc
go generate ./...
TF_ACC=1 go test ./datadog -v  -timeout 90m
=== RUN   TestProvider
--- PASS: TestProvider (0.00s)
=== RUN   TestProvider_impl
--- PASS: TestProvider_impl (0.00s)
=== RUN   TestAccDatadogMetricAlert_Basic
--- PASS: TestAccDatadogMetricAlert_Basic (3.65s)
=== RUN   TestAccDatadogOutlierAlert_Basic
--- PASS: TestAccDatadogOutlierAlert_Basic (2.46s)
=== RUN   TestAccDatadogServiceCheck_Basic
--- PASS: TestAccDatadogServiceCheck_Basic (2.49s)
PASS
ok      github.com/ojongerius/terraform-provider-datadog/datadog        8.614s
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
