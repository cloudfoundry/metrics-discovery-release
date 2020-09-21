#!/bin/bash
set -eox pipefail

function set_globals {
    pipeline_name="metrics-discovery-release"
    TARGET="${TARGET:-denver}"
    FLY_URL="https://concourse.denver.com"
}

function set_pipeline {
    pipeline_file="ci/metrics-discovery-release.yml"

    if [[ ${pipeline_file} = *.erb ]]; then
      erb ${pipeline_file} > /dev/null # this way if the erb fails the script bails
    fi

    echo setting pipeline for "$pipeline_name"

    fly -t ${TARGET} set-pipeline -p "$pipeline_name" \
        -c <(erb ${pipeline_file}) \
        -l <(lpass show 'Shared-Loggregator (Pivotal Only)/pipeline-secrets.yml' --notes) \
        -l <(lpass show 'Shared-CF- Log Cache (Pivotal ONLY)/release-credentials.yml' --notes) \
        -l <(lpass show 'Shared-Pivotal Common/pas-releng-fetch-releases' --notes)
}

function sync_fly {
    if ! fly -t ${TARGET} status; then
      fly -t ${TARGET} login -b -c ${FLY_URL}
    fi
    fly -t ${TARGET} sync
}


function main {
    set_globals $1
    sync_fly
    set_pipeline
}

lpass ls 1>/dev/null
main $1 $2
