#!/bin/bash

helpme()
{
    cat <<HELPMEHELPME

Syntax: ${0} -e <environment> -d <docker_version> -k <kubectl_version>

* All flags are optional except for -e; default values are set.

HELPMEHELPME
}

BUILD_ARGS=""

while test $# -gt 0; do
    case "$1" in 
        ""|"-?"|"-h"|"--help"|"help")
            helpme
            exit 0
            ;;
        -e)
            shift 
            if test $# -gt 0; then 
                BUILD_ARGS="$BUILD_ARGS --build-arg env_name=$1"
            else
                echo "Environment not specified"
                exit 1
            fi 
            shift 
            ;;
        -d)
            shift 
            if test $# -gt 0; then 
                BUILD_ARGS="$BUILD_ARGS --build-arg docker_version=$1"
            else
                echo "Docker version not specified"
                exit 1
            fi 
            shift
            ;; 
        -k)
            shift 
            if test $# -gt 0; then 
                BUILD_ARGS="$BUILD_ARGS --build-arg k8s_version=$1"
            else
                echo "Kubernetes not specified"
                exit 1
            fi 
            shift 
            ;;
        *)
            shift
            ;;
    esac
done

if [[ `grep -c "env_name=" <<< "$BUILD_ARGS"` -eq 0 ]]; then
    echo "ERROR: Environment required"
    helpme
    exit 1
fi

DIR="."

if command -v python >/dev/null 2>&1 ; then
    DIR=`dirname $(python -c 'import os,sys;print(os.path.realpath(sys.argv[1]))' $0)`
elif command -v python3 >/dev/null 2>&1 ; then
    DIR=`dirname $(python3 -c 'import os,sys;print(os.path.realpath(sys.argv[1]))' $0)`
elif command -v greadlink >/dev/null 2>&1 ; then
    DIR=`dirname "$(greadlink -f "$0")"`
elif command -v readlink >/dev/null 2>&1 ; then
    DIR=`dirname "$(readlink -f "$0")"`
fi

cd "$DIR/.."

set -o xtrace

# Build the docker image
docker build -f docker/Dockerfile -t api-proxy-tests $BUILD_ARGS .

# Create a docker volume to store the logs
docker volume create --name api-proxy-tests-logs
