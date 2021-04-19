#!/bin/bash
set -eox pipefail

function set_globals {
    pipeline_name="metrics-discovery-release"
    TARGET="${TARGET:-denver}"
    FLY_URL="https://concourse.cf-denver.com"
    TEAM="loggregator"
}

function set_pipeline {
    pipeline_file="ci/metrics-discovery-release.yml"

    if [[ ${pipeline_file} = *.erb ]]; then
      erb ${pipeline_file} > /dev/null # this way if the erb fails the script bails
    fi

    echo setting pipeline for "$pipeline_name"

    fly -t ${TARGET} set-pipeline -p "$pipeline_name" \
        -c <(erb ${pipeline_file}) \
        -l <(lpass show --notes 'Shared-TAS-Runtime/logging-pipeline-secrets') \
        -l <(lpass show --notes 'Shared-TAS-Runtime/release-credentials-log-cache.yml') \
        -l <(lpass show --notes 'Shared-Pivotal Common/pas-releng-fetch-releases' --notes)
}

function sync_fly {
    if ! fly -t ${TARGET} status; then
      fly -t ${TARGET} login -b -c ${FLY_URL} -n ${TEAM}
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
