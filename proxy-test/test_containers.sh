#!/bin/bash 

helpme()
{
    cat <<HELPMEHELPME

Syntax: ${0} -l <proxy_location> -n <network_id> -t <tenant_id> -k <test_kube?> -c <num_containers> -p <num_pods> -a <parallel?>

* All flags technically optional; default values are set. 

Where:
    proxy_location =
        local - test is targeting local instance of proxy (localhost:6969). With proxy running on a container, this should be the option in use. (default) 
        dev-mon01 - test is targeting remote instance (https://containers-api-dev.stage1.ng.bluemix.net:9443)

    network_id = id of network element to be inspected (currently unsupported - default is empty)

    tenant_id = tenant_id to be tested. Currently, default is "test1".

    test_kube? = "true" or "false". Default is true. 

    num_containers = total number of containers to be created; default value is 5. 

    num_pods = total number of pods to be created. If argument not given, default value is 5. 

    parallel? = "true" or "false". Default is false. 

Note: 
    when running local, specify whether autentication is done by CCSAPI:
        export DOCKER_CONFIG=certs/ccsapi 
    or local fileauth:
        export DOCKER_CONFIG=certs/fileauth

    *** If executing multiple-user fileauth, export DOCKER_CONFIG=certs/fileauth/<user_id> *** 

HELPMEHELPME
}

while test $# -gt 0; do
    case "$1" in 
        ""|"-?"|"-h"|"--help"|"help")
            helpme
            exit 1
            ;;
        -l)
            shift 
            if test $# -gt 0; then 
                PROXY_LOC=$(echo "$1" | tr '[:upper:]' '[:lower:]')
            else
                echo "Proxy location not specified"
                exit 1
            fi 
            shift 
            ;;
        -n)
            shift
            if test $# -gt 0; then 
                try_net_id="$1"
            else 
                echo "No network id specified"
                exit 1
            fi 
            shift
            ;; 
        -t)
            shift
            if test $# -gt 0; then 
                TENANT_ID="$1"
            else
                echo "No tenant_id specified"
                exit 1
            fi 
            shift 
            ;;
        -k)
            shift 
            if test $# -gt 0; then
                TEST_KUBE=$(echo "$1" | tr '[:upper:]' '[:lower:]')
            else
                echo "Test kube flag not specified"
                exit 1
            fi 
            shift 
            ;; 
        -c)
            shift
            if test $# -gt 0; then 
                NUM_CONTAINERS=$1
            else
                echo "Number of containers not specified"
                exit 1
            fi 
            shift
            ;;
        -p)
            shift
            if test $# -gt 0; then 
                NUM_PODS=$1
            else 
                echo "Number of pods not specified"
                exit 1
            fi 
            shift 
            ;;
        -a)
            shift
            if test $# -gt 0; then 
                PARALLEL="$1"
            else 
                echo "Parallel flag not specified"
                exit 1
            fi
            shift
            ;;
        *)
            shift
            ;;
    esac
done

# Setting default parameters
if [[ "$PROXY_LOC" == "" ]]; then
    PROXY_LOC="local"
fi
if [[ "$NUM_CONTAINERS" == "" ]]; then
    NUM_CONTAINERS=5
fi
if [[ "$TEST_KUBE" == "" ]]; then
    TEST_KUBE=true
fi
if [[ "$TEST_SWARM" == "" ]]; then
    TEST_SWARM=true
fi
if [[ "$NUM_PODS" == "" ]]; then
    NUM_PODS=5
fi
if [[ "$TENANT_ID" == "" ]]; then
    TENANT_ID="test1"
fi
if [[ "$PARALLEL" == "" ]]; then
    PARALLEL=false
fi

case "$PROXY_LOC" in 
    "local")
        export DOCKER_HOST=localhost:6969
        export DOCKER_TLS_VERIFY=1
        ;;
    "dev-mon01")
        export DOCKER_HOST=tcp://containers-api-dev.stage1.ng.bluemix.net:9443
        export DOCKER_TLS_VERIFY=1
        export DOCKER_CERT_PATH=certs/dev-mon01
        export DOCKER_CONFIG=certs/dev-mon01
        ;;
    *)
        export DOCKER_HOST="$PROXY_LOC"
        export DOCKER_TLS_VERIFY=1
        ;;
esac

if [[ -z "$LOGS_DIR" ]]; then
    LOGS_DIR="./logs"
fi
if [[ -z "$LOG_SUFFIX" ]]; then
    LOG_SUFFIX="$( date +%F )_$( date +%H-%M-%S )"
fi
if [[ ! -z "$LOG_PREFIX" ]]; then
    LOG_PREFIX="${LOG_PREFIX}_"
fi
LOG_PREFIX="${LOG_PREFIX}${TENANT_ID}"

function run_swarm_tests {
    if [ "$TEST_SWARM" = true ]; then
        ./lib/test_swarm_containers.sh "${LOGS_DIR}/${LOG_PREFIX}_test_swarm_results_${LOG_SUFFIX}.log" "$TENANT_ID" $NUM_CONTAINERS "$PARALLEL"
    fi
}

function run_k8s_tests {
    if [ "$TEST_KUBE" = true ]; then
        ./lib/test_kube_pods.sh "${LOGS_DIR}/${LOG_PREFIX}_test_kube_pods_results_${LOG_SUFFIX}.log" "$TENANT_ID" $NUM_PODS "$PARALLEL"
    fi
}

# Main

if [ "$PARALLEL" = true ]; then
    run_swarm_tests &
    run_k8s_tests &
    
    wait
else
    run_swarm_tests
    run_k8s_tests
fi
