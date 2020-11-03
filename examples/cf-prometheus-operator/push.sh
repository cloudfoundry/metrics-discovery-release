#!/bin/bash
set -exo pipefail

prometheus_dir=$(cd $(dirname ${BASH_SOURCE}) && pwd)

function create_security_group() {
  echo "Creating prometheus scrape security group"

  if ! cf security-group prometheus-scrape > /dev/null ; then
    cf create-security-group prometheus-scrape "${prometheus_dir}/asg.json"
  else
    cf update-security-group prometheus-scrape "${prometheus_dir}/asg.json"
  fi

  cf bind-security-group prometheus-scrape system system
}

function download_prometheus() {
  prometheus_version=$(curl -s https://api.github.com/repos/prometheus/prometheus/releases/latest | jq -r .tag_name)
  prometheus_stripped_version=${prometheus_version#v}
  prometheus_binary_url="https://github.com/prometheus/prometheus/releases/download/${prometheus_version}/prometheus-${prometheus_stripped_version}.linux-amd64.tar.gz"
  wget -qO- "$prometheus_binary_url" | tar xvz - --strip=1 prometheus-*/prometheus
}

function create_certificates() {
  mkdir -p certs
  pushd certs > /dev/null
    ca_cert_name=$(credhub find -n metric_scraper_ca --output-json | jq -r .credentials[].name | grep cf)
    credhub generate -n prometheus_scrape_tls -t certificate --ca "$ca_cert_name" -c prometheus_scrape_tls

    credhub get -n prometheus_scrape_tls --output-json | jq -r .value.ca > scrape_ca.crt
    credhub get -n prometheus_scrape_tls --output-json | jq -r .value.certificate > scrape.crt
    credhub get -n prometheus_scrape_tls --output-json | jq -r .value.private_key > scrape.key
  popd > /dev/null
}

function push_prometheus() {
  cf target -o system -s system

  GOOS=linux go build -o confgen
  cf v3-create-app prometheus
  cf set-env prometheus NATS_HOSTS "$(bosh instances --column Instance --column IPs | grep nats | awk '{print $2}')"

  nats_cred_name=$(credhub find --name-like nats_password --output-json | jq -r .credentials[0].name)
  cf set-env prometheus NATS_PASSWORD "$(credhub get --name ${nats_cred_name} --quiet)"

  cf v3-apply-manifest -f "${prometheus_dir}/manifest.yml"
  cf v3-push prometheus
}

pushd "${prometheus_dir}" > /dev/null
  download_prometheus
  create_security_group
  create_certificates
  push_prometheus
popd > /dev/null
