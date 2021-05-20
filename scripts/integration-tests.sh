#!/bin/bash
set -uo pipefail

router_ip=$(bosh -d cf --column Instance --column IPs vms | grep '^router' | head -n 1 | awk '{print $2}')
windows_cell_ip=$(bosh -d cf --column Instance --column IPs vms | grep '^windows' | head -n 1 | awk '{print $2}')
METRICS_AGENT_PORT=14726

bosh -d cf ssh doppler/0 -c \
  "curl https://${router_ip}:${METRICS_AGENT_PORT}/metrics -k --cert /var/vcap/jobs/metrics-agent/config/certs/scrape.crt --key /var/vcap/jobs/metrics-agent/config/certs/scrape.key --cacert /var/vcap/jobs/metrics-agent/config/certs/scrape_ca.crt" \
  > converted_metrics_results

bosh -d cf ssh doppler/0 -c \
  "curl https://${router_ip}:${METRICS_AGENT_PORT}/metrics?id=prom_scraper -k --cert /var/vcap/jobs/metrics-agent/config/certs/scrape.crt --key /var/vcap/jobs/metrics-agent/config/certs/scrape.key --cacert /var/vcap/jobs/metrics-agent/config/certs/scrape_ca.crt" \
  > prom_metrics_results

# TODO: Figure out how to curl metrics endpoints in Windows
# Powershell/Commandline

# bosh -d cf ssh windows2019-cell/0 -c \
#   "curl https://${windows_cell_ip}:${METRICS_AGENT_PORT}/metrics -k --cert /var/vcap/jobs/metrics-agent/config/certs/scrape.crt --key /var/vcap/jobs/metrics-agent/config/certs/scrape.key --cacert /var/vcap/jobs/metrics-agent/config/certs/scrape_ca.crt" \
#   > converted_metrics_results_windows

# bosh -d cf ssh windows2019-cell/0 -c \
#   "curl https://${windows_cell_ip}:${METRICS_AGENT_PORT}/metrics?id=prom_scraper -k --cert /var/vcap/jobs/metrics-agent/config/certs/scrape.crt --key /var/vcap/jobs/metrics-agent/config/certs/scrape.key --cacert /var/vcap/jobs/metrics-agent/config/certs/scrape_ca.crt" \
#   > prom_metrics_results_windows

exit_status=0

converted_metrics=(buffered_messages) # This is a gorouter metric
linux_metrics=(failed_scrapes_total forwarder_agent metrics-agent metrics_discovery_registrar metron syslog_agent udp_forwarder)
windows_metrics=() # TODO: fill out windows metrics
missing_metrics=()
found_metrics=()

function check_metric() {
    local metric=$1
    local file=$2

    grep $metric $file > /dev/null
    if [ $? -ne 0 ]; then
        missing_metrics+=($metric)
        exit_status=1
    else
        found_metrics+=($metric)
    fi
}

echo "Checking for converted Loggregator envelope metrics in Linux"
for metric in ${converted_metrics[@]}; do
    check_metric $metric converted_metrics_results
    # check_metric $metric converted_metrics_results_windows
done


echo "Checking for logging/metrics components emitting Prometheus metrics in Linux"
for metric in ${linux_metrics[@]}; do
    check_metric $metric prom_metrics_results
    # check_metric $metric prom_metrics_results_windows
done

if [[ ${exit_status} -ne 0 ]]; then
  printf '\nMissing the following metrics:\n'
  printf '\t%s\n' "${missing_metrics[@]}"

  echo "***************************************  Metrics Results  **************************************************"
  echo "*********************************** Converted Loggregator Env Metrics **************************************"
  cat converted_metrics_results

  printf "\n\n"
  echo "*********************************  Prometheus Metrics Results  *********************************************"
  cat prom_metrics_results


  # printf "\n\n"
  # echo "******************************** Converted Loggregator Env Metrics Windows *********************************"
  # cat converted_metrics_results_windows

  # printf "\n\n"
  # echo "******************************  Prometheus Metrics Results Windows *****************************************"
  # cat prom_metrics_results_windows
else
    printf '\nFound the following metrics:\n'
    printf '\t%s\n' "${found_metrics[@]}"
fi

exit $exit_status
