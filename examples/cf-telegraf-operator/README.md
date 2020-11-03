# CF-pushable Telegraf

## Overview
This repo provides a reference architecture allowing Cloud Foundry users to leverage
[agent based architecture](https://github.com/cloudfoundry/metrics-discovery-release/tree/develop/docs) for collecting
metrics. It is provided for reference only and is not intended to be used in production.

This examples utilizes the [metrics-discovery-release](https://github.com/cloudfoundry/metrics-discovery-release)
to gather all the component metrics endpoints published to the NATs queue
"metrics.scrape_targets".

## Minimum Requirements
* BOSH CLI v6.1.1
* CF CLI v6.47.0
* CF Deployment v12.21.0
* Credhub CLI version 2.5.3

## Components
There are two components: `telegraf` and `telegraf-config-sidecar`.

The `telegraf-config-sidecar` will generate the Prometheus input config from the
metrics endpoints it gets from the NATs queue and add that to the telegraf configuration.
The NATs queue is checked every 15 seconds to see if any updates are needed to the input config.
Every 45 seconds, the the sidecar restarts telegraf to pick up these new metrics endpoints.

## Usage
1. Add output plugin(s) to telegraf.conf
1. `cf`, `credhub`, and `bosh` target the desired environment
1. `./push.sh`

## Scaling
Scaling of telegraf can be handled directly by Diego. That said, this implementation will not
ensure "only once" delivery. Scaling to two instances will result in duplicate metrics, three
instances will triple the metrics etc.

#### Security Group Restrictions
Due to application security groups, Telegraf cannot scrape the Diego Cell it is running on.
This means there must be at least 2 instances of Telegraf (on different diego cells) in
order to ingest all metrics.

#### Dropping metrics
This promQL query will allow you to determine if a specific output is not keeping up.
A good number to shoot for is 99% of metrics getting through.
Just replace `my-output-plugin` with the name of your output e.g. datadog.
```
100 * (1 -
  rate(internal_write_metrics_dropped{output="my-output-plugin"}[1m]) /
  rate(internal_write_metrics_written{output="my-output-plugin"}[1m]))
```

If this number is below 99%, try increasing the `metric_buffer_limit`.
