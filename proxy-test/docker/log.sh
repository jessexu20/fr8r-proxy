#!/bin/bash

helpme()
{
    cat <<HELPMEHELPME

Syntax: ${0} -t <log_tag> --latest

HELPMEHELPME
}

LOG_TAG=""
LATEST=0

while test $# -gt 0; do
    case "$1" in 
        ""|"-?"|"-h"|"--help"|"help")
            helpme
            exit 0
            ;;
        "--latest")
            shift
            LATEST=1
            ;;
        -t)
            shift
            if test $# -gt 0; then 
                LOG_TAG="$1"
            else
                echo "No log tag specified"
            fi 
            shift 
            ;;
        *)
            shift
            ;;
    esac
done

# -t or --latest must be passed
if [[ -z "$LOG_TAG" && $LATEST -eq 0 ]]; then
    helpme
    exit 1
fi

# If --latest get the latest log tag
LOGS_DIR=`docker inspect -f "{{ .Config.Env }}" api-proxy-tests | grep -oP "(?<=LOGS_DIR\=)([^\s\]]+)"`
if [[ $LATEST -eq 1 ]]; then
    LOG_TAG=`docker run -v api-proxy-tests-logs:"$LOGS_DIR" api-proxy-tests --list-logs | tail -n 1`
fi

# If LOG_TAG is empty there aren't any logs
if [[ -z "$LOG_TAG" ]]; then
    echo "No log found"
    exit 1
fi

# Print the log with tag LOG_TAG
docker run -v api-proxy-tests-logs:"$LOGS_DIR" api-proxy-tests -s "$LOG_TAG"
