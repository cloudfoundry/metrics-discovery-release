## Overview
The Log and Metric Agent Architecture (Experimental) contains a group of components that allow you to access all the same 
logs and metrics that you can access through the Loggregator system. The components of the Log and 
Metric Agent Architecture use a shared-nothing architecture that requires several fewer VMs than the
Loggregator system.

The Log and Metric Agent Architecture includes components that collect, store, and forward logs and metrics
in your Pivotal Platform deployment

### Metric Components
This section describes the components of the Log and Metric Agent Architecture that allow you to access metrics for your foundation.

These components allow you to access the same metrics available through the Loggregator Firehose with a pull-based architecture. The Loggregator system uses a push-based model for forwarding metrics, in which all data is sent though the Firehose.

The following components of the Log and Metric Agent Architecture enable pull-based access to metrics:

 * Metrics Agent ([Loggregator Agent](https://github.com/cloudfoundry/loggregator-agent-release)):
The Metrics Agent collects Loggregator V2 envelopes and makes them available on a Prometheus endpoint. The Metrics Agent performs a similar function to the Loggregator Agent in the Loggregator system.

 * Metrics Discovery Registrar:
The Metrics Discovery Registrar publishes the location of the Prometheus endpoint defined by the Metrics Agent and Service Metrics Agent to NATs. This is helpful for configuring automation to scrape metric data from the endpoint. For more information about automating metric scraping, see [Reference Architectures](#Reference-Architectures).

### Log Components 
This section describes the Log and Metric Agent Architecture components that allow you to access logs on your foundation.

These components are also a part of the Loggregator system. For more information about how these components function as part of the Loggregator system, see Loggregator Architecture.

The following components of the Log and Metric Agent Architecture enable access to logs:

* Syslog Agent ([Loggregator Agent](https://github.com/cloudfoundry/loggregator-agent-release)):
Syslog Agents run on Pivotal Platform component VMs and host VMs to collect and forward logs to configured syslog drains. This includes syslog drains for individual apps as well as aggregate drains for all apps in your foundation. You can specify the destination for logs as part of the individual syslog drain or in the PAS tile.

* Aggregate Syslog Drain:
The aggregate syslog drain feature allows you to configure all Syslog Agents on your deployment to send logs to a single destination. You can use the aggregate syslog drain feature rather than the Loggregator Firehose to forward all logs for your deployment.

* [Log Cache](https://github.com/cloudfoundry/log-cache): Log Cache allows you to view logs and metrics over a specified period of time. The Log Cache includes API endpoints and a CLI plugin to query and filter logs and metrics. To download the Log Cache CLI plugin, see Cloud Foundry Plugins. The Log Cache API endpoints are available by default. For more information about using the Log Cache API, see Log Cache on GitHub.

### Reference Architectures
The following reference architectures are intended for commercial vendors to build upon for thier own commercial offerings. They are not intended to be final production ready integrations.

* [cf telegraf operator](https://github.com/cloudfoundry-incubator/cf-telegraf-operator) - This leverages telegraf to convery a pull based metric system into a push based system used by many existing observability tools
* [cf prometheus operator](https://github.com/cloudfoundry-incubator/cf-prometheus-operator) - This example shows how you can leverage NATs service discovery and pull based metrics to cf push a prometheus instances that collects all platform metrics. 


