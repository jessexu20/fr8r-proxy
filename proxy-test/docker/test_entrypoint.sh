#!/bin/bash

helpme()
{
    cat <<HELPMEHELPME

Run tests: -l <proxy_location> -n <network_id> -k <test_kube?> -a <parallel?> -c <num_containers> -p <num_pods> -t <shard>:<tenant_id>:<api_key>
* All flags are optional except for -t; default values are set.

Show log: -s <log_tag>
* All flags are required

List logs: --list-logs

HELPMEHELPME
}

function get_shard {
     echo "$1" | cut -d: -f 1
}

function get_tenant_id {
     echo "$1" | cut -d: -f 2
}

function get_api_key {
     echo "$1" | cut -d: -f 3
}

DIR="$(dirname "$(readlink -f "$0")")"
RUN_TEST="$DIR/test_containers.sh"
TENANTS=""

if [ "$LOGS_DIR" = "" ] ; then
    LOGS_DIR="$DIR/logs"
fi

if [ "$CERTS_DIR" = "" ] ; then
    CERTS_DIR="$DIR/certs"
fi

if [ "$ENV_NAME" = "" ] ; then
    echo "Environment name not found, please set a value for ENV_NAME"
    exit 1
fi

CERTS_DIR="$CERTS_DIR/$ENV_NAME"

while test $# -gt 0; do
    case "$1" in 
        ""|"-?"|"-h"|"--help"|"help")
            helpme
            exit 1
            ;;
        "--list-logs")
            ls "$LOGS_DIR" | rev | cut -d. -f2- | cut -d_ -f1,2 | rev | sort -u
            exit 0
            ;;
        -s)
            shift
            ls "$LOGS_DIR" | grep $1 | while read -r file ; do
                echo
                echo "$file:"
                cat "$LOGS_DIR/$file"
                echo
            done
            exit 0
            ;;
        -l)
            shift 
            if test $# -gt 0; then 
                RUN_TEST="$RUN_TEST -l $(echo "$1" | tr '[:upper:]' '[:lower:]')"
            else
                echo "Proxy location not specified"
                exit 1
            fi 
            shift 
            ;;
        -n)
            shift
            if test $# -gt 0; then 
                RUN_TEST="$RUN_TEST -n $1"
            else 
                echo "No network id specified"
                exit 1
            fi 
            shift
            ;; 
        -t)
            shift
            if test $# -gt 0; then 
                TENANTS="$TENANTS;$1"
            else
                echo "No tenant_id specified"
                exit 1
            fi 
            shift 
            ;;
        -k)
            shift 
            if test $# -gt 0; then
                RUN_TEST="$RUN_TEST -k $(echo "$1" | tr '[:upper:]' '[:lower:]')"
            else
                echo "Test kube flag not specified"
                exit 1
            fi 
            shift 
            ;; 
        -c)
            shift
            if test $# -gt 0; then 
                RUN_TEST="$RUN_TEST -c $1"
            else
                echo "Number of containers not specified"
                exit 1
            fi 
            shift
            ;;
        -p)
            shift
            if test $# -gt 0; then 
                RUN_TEST="$RUN_TEST -p $1"
            else 
                echo "Number of pods not specified"
                exit 1
            fi 
            shift 
            ;;
        -a)
            shift
            if test $# -gt 0; then 
                RUN_TEST="$RUN_TEST -a $1"
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

if [ "$TENANTS" = "" ] ; then
    helpme
    exit 1
fi

export LOG_SUFFIX="$( date +%F )_$( date +%H-%M-%S )"
echo "The env is $ENV_NAME"
echo "The log tag is $LOG_SUFFIX"

COUNT=0
TOTAL=$(echo "$TENANTS" | grep -o ";" | wc -l)

function test_completed {
    let COUNT++
    echo -e "\e[1A\e[0KCompleted: $COUNT/$TOTAL"
    wait
}
echo -e "Completed: $COUNT/$TOTAL"

trap test_completed SIGUSR1

TENANTS=$(echo "$TENANTS" | cut -c 2-)

export IFS=";"
for tenant in $TENANTS; do
    (
        shard=`get_shard "$tenant"`
        tenant_id=`get_tenant_id "$tenant"`
        api_key=`get_api_key "$tenant"`
        
        export LOG_PREFIX="$shard"
        export DOCKER_CERT_PATH="$CERTS_DIR/$shard/$api_key"
        export KUBECONFIG="$CERTS_DIR/$shard/$api_key/kube-config"

        RUN_TEST="$RUN_TEST -t $tenant_id"
        eval $RUN_TEST > /dev/null 2>&1
        
        kill -USR1 $$
    ) &
done

wait

TOTAL_FAILED=0

for tenant in $TENANTS; do
    LOG_PREFIX=`get_shard "$tenant"`
    tenant_id=`get_tenant_id "$tenant"`
    
    SWARM_LOG="$LOGS_DIR/${LOG_PREFIX}_${tenant_id}_test_swarm_results_${LOG_SUFFIX}.log"
    KUBE_LOG="$LOGS_DIR/${LOG_PREFIX}_${tenant_id}_test_kube_pods_results_${LOG_SUFFIX}.log"
    
    echo
    echo "Test results for tenant ${tenant_id}:"
    [[ -f "$SWARM_LOG" ]] && grep -A4 ^Docker "$SWARM_LOG"
    [[ -f "$KUBE_LOG"  ]] && grep -A4 ^Kube "$KUBE_LOG"
    
    FAILED_SWARM=0
    FAILED_KUBE=0
    [[ -f "$SWARM_LOG" ]] && FAILED_SWARM=`grep -oP "(?<=Failed \= )([0-9]+)" "$SWARM_LOG"`
    [[ -f "$KUBE_LOG"  ]] && FAILED_KUBE=`grep -oP "(?<=Failed \= )([0-9]+)" "$KUBE_LOG"`
    TOTAL_FAILED=$((TOTAL_FAILED + FAILED_SWARM + FAILED_KUBE))
done

echo -e "\n$TOTAL_FAILED failed tests.\n"

exit $TOTAL_FAILED
