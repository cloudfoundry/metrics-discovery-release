## Using Metrics Agent
Metrics Agent converts Loggregator metrics from the Forwarder Agent into Prometheus Exposition style metrics and hosts
them on a Prometheus-scrapable endpoint. Loggregator v2 Envelopes are converted to Prometheus metric types.

#### Conversion
| Loggregator envelope type                                   | Prometheus type                                                                                                                                                                                   |
|-------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Counter <br> - Integer                                      | Counter <br> - Float                                                                                                                                                                              |
| Gauge <br> - Potentially many distinct metrics per envelope | Gauge(s) <br> - Potentially many per Loggregator envelope                                                                                                                                         |
| Timers (http metrics only)                                  | Histogram <br> - Tags are collapsed into a label set based on the `metrics.whitelisted_timer_tags` property <br> - Values recorded are the difference between the start and stop times in seconds |               
               
### Deploying
               
#### Standalone

To deploy metrics agent standalone, add the following job to all instance groups and the variables to the variables section.
If metrics from metrics agent are desired, deploy prom scraper with the same `metric_scraper_ca`
 
 ```yaml
 jobs:
 - name: metrics-agent
   release: metrics-discovery
   properties:
     grpc:
       ca_cert: "((loggregator_tls_agent.ca))"
       cert: "((loggregator_tls_agent.certificate))"
       key: "((loggregator_tls_agent.private_key))"
     metrics:
       ca_cert: "((metrics_agent_tls.ca))"
       cert: "((metrics_agent_tls.certificate))"
       key: "((metrics_agent_tls.private_key))"
       server_name: metrics_agent
     scrape:
       tls:
         ca_cert: "((prom_scraper_scrape_tls.ca))"
         cert: "((prom_scraper_scrape_tls.certificate))"
         key: "((prom_scraper_scrape_tls.private_key))"
 variables:
 - name: loggregator_tls_agent
   type: certificate
   options:
     ca: /bosh-<ENV_NAME>/cf/loggregator_ca
     common_name: metron
     extended_key_usage:
     - client_auth
     - server_auth
 - name: metric_agent_tls
   type: certificate
   options:
     ca: metric_scraper_ca
     common_name: metrics_agent
     extended_key_usage:
     - server_auth
 - name: metric_scraper_ca
   type: certificate
   options:
     is_ca: true
     common_name: metricScraperCA
 ```
 
 ##### With Forwarder Agent
 
 To deploy metrics agent downstream of a Forwarder Agent, add the following jobs to all instance groups and the variables to the variables section.
 If metrics from metrics agent are desired, deploy prom scraper with the same `metric_scraper_ca`
 
 ```yaml
 jobs:
 - name: metrics-agent
   release: metrics-discovery
   properties:
     grpc:
       ca_cert: "((loggregator_tls_agent.ca))"
       cert: "((loggregator_tls_agent.certificate))"
       key: "((loggregator_tls_agent.private_key))"
     metrics:
       ca_cert: "((metrics_agent_tls.ca))"
       cert: "((metrics_agent_tls.certificate))"
       key: "((metrics_agent_tls.private_key))"
       server_name: metrics_agent
     scrape:
       tls:
         ca_cert: "((prom_scraper_scrape_tls.ca))"
         cert: "((prom_scraper_scrape_tls.certificate))"
         key: "((prom_scraper_scrape_tls.private_key))"
 - name: forwarder_agent
   include:
     stemcell:
     - os: ubuntu-xenial
   jobs:
   - name: loggr-forwarder-agent
     release: loggregator-agent
     properties:
       tls:
         ca_cert: "((loggregator_tls_agent.ca))"
         cert: "((loggregator_tls_agent.certificate))"
         key: "((loggregator_tls_agent.private_key))"
       metrics:
         ca_cert: "((forwarder_agent_metrics_tls.ca))"
         cert: "((forwarder_agent_metrics_tls.certificate))"
         key: "((forwarder_agent_metrics_tls.private_key))"
         server_name: forwarder_agent_metrics
 variables:
 - name: loggregator_tls_agent
    type: certificate
   options:
     ca: /bosh-<ENV_NAME>/cf/loggregator_ca
     common_name: metron
     extended_key_usage:
     - client_auth
     - server_auth
 - name: metric_agent_tls
    type: certificate
    options:
      ca: metric_scraper_ca
      common_name: metrics_agent
      extended_key_usage:
      - server_auth
 - name: forwarder_agent_metrics_tls
   type: certificate
   options:
     ca: metric_scraper_ca
     common_name: forwarder_agent_metrics
     extended_key_usage:
     - server_auth
 - name: metric_scraper_ca
   type: certificate
   options:
     is_ca: true
     common_name: metricScraperCA
 ```
