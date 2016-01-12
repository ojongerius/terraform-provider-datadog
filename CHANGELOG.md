## 0.0.5 (unreleased)
IMPROVEMENTS:
  * add support for warning and critical values per monitor
  * detect pre 0.0.4 resources, and allow conversion by exporting TF_YOLO (use at your own risk)

## 0.0.4 (Jan 12, 2016)
FEATURES:
  * datadog_outlier_alert

CHANGES:
  * removal of datadog_monitor, datadog_graph, datadog_dashboard
  * each resource now generates *one* resource per count. This is a breaking change,
    old metric_alert resources are not supported.

## 0.0.3 (unreleased)
FEATURES:

  * datadog_metric_alert

## 0.0.2 (Sep 28, 2015)
FEATURES:

  * datadog_service_check

## 0.0.1 (Aug 16, 2015)

  * Initial release
