#!/bin/bash
set -eo pipefail

bosh -d cf ssh doppler -c 'sudo rm -f /tmp/prom*' &
P0=$!

set +e
bosh -d cf ssh doppler -c 'ps aux | grep prometheus | grep -v grep' > /dev/null 2>&1 && echo "Prometheus is already running" && exit 1
set -e

cd ${HOME}/workspace/go
GO111MODULE=off go get github.com/prometheus/prometheus/...
cd src/github.com/prometheus/prometheus/cmd

pushd prometheus > /dev/null
    GOOS=linux go build .
popd > /dev/null

pushd promtool > /dev/null
    GOOS=linux go build .
popd > /dev/null

wait $P0

bosh -d cf scp ./prometheus/prometheus doppler:/tmp &
P1=$1
bosh -d cf scp ./promtool/promtool doppler:/tmp &
P2=$!
wait $P1 $P2

bosh -d cf ssh doppler -c 'sudo mv /tmp/prom* /root && sudo chmod +x /root/prometheus /root/promtool && \
cat << EOF > /tmp/prom.yml
---
global:
  scrape_interval: 15s
scrape_configs:
- job_name: cf
  file_sd_configs:
  - files:
    - /var/vcap/data/scrape-config-generator/scrape_targets.json
  scheme: https
  tls_config:
    ca_file: /var/vcap/jobs/prom_scraper/config/certs/scrape_ca.crt
    cert_file: /var/vcap/jobs/prom_scraper/config/certs/scrape.crt
    key_file: /var/vcap/jobs/prom_scraper/config/certs/scrape.key
    server_name: metrics_agent
EOF'

echo "Starting prometheus... don't end ssh session"
bosh -d cf ssh doppler -c 'sudo mv /tmp/prom.yml /root && sudo /root/prometheus --storage.tsdb.path /root/promdata --web.listen-address "127.0.0.1:6000" --config.file /root/prom.yml'
