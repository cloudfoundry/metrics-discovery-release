## CF-pushable Prometheus server

The `./push.sh` sets up the appropriate security groups to allow scraping for
Prometheus as well as pushing the Prometheus server.

After pushing the app, visit the Prometheus dashboard at
`prometheus.<system_domain>/graph`.
