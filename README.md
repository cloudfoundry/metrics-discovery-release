# Metrics Discovery Release
[![slack.cloudfoundry.org][slack-badge]][loggregator-slack]
[![CI Badge][ci-badge]][ci-pipeline]

The Metrics Discovery Release is intended to give release authors an easy way to discover Prometheus Exposition formatted
metrics endpoints in their Cloudfoundry Deployments. It consists of two components: The Metrics Discovery Registrar and
the Scrape Config Generator.

![architecture]

## Metrics Discovery Registrar

The Metrics Discovery Registrar publishes scrape configs to CF NATS to be consumed by a Scrape Config Generator.

### Configuration
Interested jobs should provide a `metric_targets.yml` file and place it in the location described by the `targets_glob`
property -- by default `var/vcap/data/*/metric_targets.yml`.

The `metric_targets.yml` should provide information
about the endpoint to be scraped using the [Prometheus format](https://prometheus.io/docs/prometheus/latest/configuration/configuration/).

[Example `metric_targets.yml`][target-example]

## Scrape Config Generator

The Scrape Config Generator subscribes to CF NATS and consumes published metric targets. It aggregates the metric targets
and saves a Prometheus-formatted [scrape config](https://prometheus.io/docs/prometheus/latest/configuration/configuration/)
to the path defined by the `scrape_config_file_path` property -- by default `/var/vcap/data/scrape-config-generator/scrape_configs.yml`

The scrape config will be modified as metric targets come and go. Interested metric scrapers should watch the scrape config file
for changes.

### Metrics agent
An agent that proxies to components with a `prom_scraper_config.yml` and
receives metrics from the Forwarder Agent and exposes them on a prometheus-scrapable endpoint.
More information can be found in the [docs][metrics-agent]

### CI Pipeline
The configuration for the Concourse pipeline lives in the `ci` directory. To
fly and set up the pipeline, use the `./ci/set-pipeline.sh` script. This will
set `ci/metrics-discovery-release.yml` pipeline config.

[slack-badge]:         https://slack.cloudfoundry.org/badge.svg
[loggregator-slack]:   https://cloudfoundry.slack.com/archives/loggregator
[ci-badge]:            https://concourse.cf-denver.com/teams/loggregator/pipelines/metrics-discovery-release/badge
[ci-pipeline]:         https://concourse.cf-denver.com/teams/loggregator/pipelines/metrics-discovery-release

[metrics-agent]:        docs/metrics-agent.md
[architecture]:         docs/metrics_discovery_release_architecture.png
[target-example]:       docs/metric_targets.yml
