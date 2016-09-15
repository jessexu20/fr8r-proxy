#!/bin/bash
# set -v

function build_proxy_image {
    local name="$1"
    local context="$2"
    shift 2
    
    # build the docker image
    docker build "$@" -t "$name" "$context"
}

build_proxy_image "api-proxy" "." "$@"
