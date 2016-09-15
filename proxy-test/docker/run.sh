#!/bin/bash

function main {
    local CERTS_DIR=`docker_image_env api-proxy-tests CERTS_DIR`
    local LOGS_DIR=`docker_image_env api-proxy-tests LOGS_DIR`
    local ENV_NAME=`docker_image_env api-proxy-tests ENV_NAME`
    local ENVS_DIR="$HOME/.fr8r/envs"

    local tenants=()
    if [[ `echo "$CI" | tr '[:upper:]' '[:lower:]'` == true && `check_for_tenants "$@"` == false ]]; then
        local CREDS_DIR="$ENVS_DIR/$ENV_NAME/creds.json"
        IFS=$'\n' tenants=(`cat "$CREDS_DIR" \
            | grep -oP "((?<=\"TLS_path\":\")|(?<=\"Space_id\":\"))[^\"]+" \
            | rev \
            | sed 'N;s/\(.*\)\n\([^/]*\)\(\/\)\([^/]*\)\(.*\)/t-\n\2\:\1\:\4/' \
            | rev`)
    fi

    docker run -v "$ENVS_DIR":"$CERTS_DIR":ro -v api-proxy-tests-logs:"$LOGS_DIR" --net="host" api-proxy-tests "${tenants[@]}" "$@"
}

function docker_image_env {
    docker inspect -f "{{ .Config.Env }}" "$1" | grep -oP "(?<=$2\=)([^\s\]]+)"
}

function check_for_tenants {
    local have_tenant=false
    
    while [[ $# -gt 0 && $have_tenant == false ]]; do
        case "$1" in 
        -t)
            shift
            have_tenant=true
            ;;
        *)
            shift
            ;;
        esac
    done
    
    echo "$have_tenant"
}

main "$@"
