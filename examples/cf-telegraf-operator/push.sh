#!/bin/bash
set -exo pipefail

telegraf_operator_dir=$(cd $(dirname ${BASH_SOURCE}) && pwd)

function create_security_group() {
  echo "Creating Telegraf scrape security group"

  if ! cf security-group telegraf-scrape > /dev/null ; then
    cf create-security-group telegraf-scrape "${telegraf_operator_dir}/asg.json"
  else
    cf update-security-group telegraf-scrape "${telegraf_operator_dir}/asg.json"
  fi

  cf bind-security-group telegraf-scrape system system
}

function download_telegraf() {
  telegraf_version=$(curl -s https://api.github.com/repos/influxdata/telegraf/releases/latest | jq -r .tag_name || "1.15.3")
  telegraf_version_stripped=${telegraf_version#v}
  platform="linux"
  arch="amd64"
  telegraf_binary_url="https://dl.influxdata.com/telegraf/releases/telegraf-${telegraf_version_stripped}_${platform}_${arch}.tar.gz"
  wget -O telegraf.tar.gz "$telegraf_binary_url"

  # This only grabs the telegraf binary
  # It is very dependent on the archive directory structure and may not live
  # here in future versions
  tar xvf telegraf.tar.gz --strip=4 telegraf-${telegraf_version_stripped}/usr/bin/telegraf
}

function create_certificates() {
  mkdir -p certs
  pushd certs > /dev/null
    ca_cert_name=$(credhub find -n metric_scraper_ca --output-json | jq -r .credentials[].name | grep cf)
    credhub generate -n telegraf_scrape_tls -t certificate --ca "$ca_cert_name" -c telegraf_scrape_tls

    credhub get -n telegraf_scrape_tls --output-json | jq -r .value.ca > scrape_ca.crt
    credhub get -n telegraf_scrape_tls --output-json | jq -r .value.certificate > scrape.crt
    credhub get -n telegraf_scrape_tls --output-json | jq -r .value.private_key > scrape.key
  popd > /dev/null
}

function push_telegraf() {
  GOOS=linux go build -o confgen
  cf v3-create-app telegraf
  cf set-env telegraf NATS_HOSTS "$(bosh instances --column Instance --column IPs | grep nats | awk '{print $2}')"

  nats_cred_name=$(credhub find --name-like nats_password --output-json | jq -r .credentials[0].name)
  cf set-env telegraf NATS_PASSWORD "$(credhub get --name ${nats_cred_name} --quiet)"

  cf v3-apply-manifest -f "${telegraf_operator_dir}/manifest.yml"
  cf v3-push telegraf
}


function create_and_target_space_and_org() {
  cf create-space system -o system
  cf target -o system -s system
}

pushd ${telegraf_operator_dir} > /dev/null
  download_telegraf
  create_and_target_space_and_org
  create_security_group
  create_certificates
  push_telegraf
popd > /dev/null
