#!/bin/bash

DIR=`dirname "$0"`
source "$DIR/common_test_func.sh"

function main() {
    local LOGS_PATH="$1"
    local TENANT_ID="$2"
    local PARALLEL="$4"
    local NUM_PODS=$3
    local KUBE_NAME="${TENANT_ID}-kube-web-test"
    local TEST_ID="$( date +%Y%m%d )$( date +%H%M%S )"

    setup_env
    
    init_tests "$TENANT_ID" "kube" "$LOGS_PATH"

    test_kubectl

    local COUNTER=1
    while [ $COUNTER -le $NUM_PODS ]; do
    
        if [[ $PARALLEL == true ]]; then
            test_pod_creation "$TEST_ID" "$KUBE_NAME" "$TENANT_ID" $COUNTER &
            increment_test_count
        else
            test_pod_creation "$TEST_ID" "$KUBE_NAME" "$TENANT_ID" $COUNTER
        fi

        let COUNTER++
    done
    
    wait
    
    COUNTER=1
    while [ $COUNTER -le $NUM_PODS ]; do
    
        if [[ $PARALLEL == true ]]; then
            test_pod_runtime "$TEST_ID" "$KUBE_NAME" "$TENANT_ID" $COUNTER &
            increment_test_count
        else
            test_pod_runtime "$TEST_ID" "$KUBE_NAME" "$TENANT_ID" $COUNTER
        fi

        let COUNTER++
    done
    
    wait

    COUNTER=1
    while [ $COUNTER -le $NUM_PODS ]; do
    
        if [[ $PARALLEL == true ]]; then
            test_pod_deletion "$TEST_ID" "$KUBE_NAME" "$TENANT_ID" $COUNTER &
            increment_test_count
        else
            test_pod_deletion "$TEST_ID" "$KUBE_NAME" "$TENANT_ID" $COUNTER
        fi

        let COUNTER++
    done
    
    wait

    complete_tests "Kube"
}

# Tests

function test_kubectl {
    begin_test_block
    
    assert_get_pods 0
    assert_get_pods_namespaces 1
    
    assert_get_nodes 1
    
    end_test_block
}

function test_pod_creation {
    local TEST_ID="$1"
    local KUBE_NAME="$2"
    local TENANT_ID="$3"
    local COUNTER="$4"
    
    local kube_name="${KUBE_NAME}${COUNTER}"
    local yaml_path=`get_yaml_path "$TENANT_ID" $COUNTER`

    make_yaml "$TEST_ID" $COUNTER "$TENANT_ID" "$yaml_path"
    delete_if_exists "$kube_name"
    
    begin_test_block
    
    assert_create_pod "$yaml_path" 0
    assert_describe_pod "$kube_name" 0
    assert_running "$kube_name" "$TEST_ID"
    
    end_test_block
}

function test_pod_runtime {
    local TEST_ID="$1"
    local KUBE_NAME="$2"
    local TENANT_ID="$3"
    local COUNTER="$4"
    
    local kube_name="${KUBE_NAME}${COUNTER}"
    local yaml_path=`get_yaml_path "$TENANT_ID" $COUNTER`
    
    begin_test_block

    assert_running "$kube_name" "$TEST_ID"
    assert_create_pod "$yaml_path" 1
    assert_log "$kube_name" 0
    
    end_test_block
}

function test_pod_deletion {
    local TEST_ID="$1"
    local KUBE_NAME="$2"
    local TENANT_ID="$3"
    local COUNTER="$4"
    
    local kube_name="${KUBE_NAME}${COUNTER}"
    
    begin_test_block

    assert_delete_pod "$kube_name" 0
    assert_not_running "$kube_name" "$TEST_ID"
    
    end_test_block
}

function assert_get_nodes {
    assert kubectl get nodes \
           --equal $1 \
           --log "get nodes"
}

function assert_get_pods {
    assert kubectl get pods \
           --equal $1 \
           --log "get pods"
}

function assert_running {
    assert kubectl get pods --selector="test_id=$2" \
           --equal 0 \
           --output-contains "$1" \
           --log "get pods"
}

function assert_not_running {
    assert kubectl get pods --selector="test_id=$2" \
           --equal 0 \
           --output-not-contains "$1" \
           --log "get pods"
}

function assert_get_pods_namespaces {
    assert kubectl get pods --all-namespaces \
           --equal $1 \
           --log "get pods in all namespaces"
}

function assert_create_pod {
    assert kubectl create -f "$1" \
           --equal $2 \
           --log "create pod; pod = $1"
}

function assert_describe_pod {
    assert kubectl describe pod "$1" \
           --equal $2 \
           --log "describe pod; pod = $1"
}

function assert_delete_pod {
    assert kubectl delete pod --now=true "$1" \
           --equal $2 \
           --log "delete pod; pod = $1"
}

function assert_log {
    assert kubectl logs --tail=1 "$1" \
           --equal $2 \
           --log "logs pod; pod = $1"
}

# Utils

function setup_env {
    local PLATFORM=`uname -s | tr '[:upper:]' '[:lower:]'`
    local KUBE_DIR="$DIR/kube"
    local KUBE_PATH="$KUBE_DIR/kubectl"
    export PATH="$KUBE_DIR:$PATH"

    if command -v kubectl >/dev/null 2>&1 ; then
        echo "Using system kubectl..."
    elif [[ ! -f "$KUBE_PATH" ]]; then
        mkdir -p "$KUBE_DIR"
        local LATEST_KUBECTL=`curl -L https://storage.googleapis.com/kubernetes-release/release/stable.txt`
        curl -sSLo "$KUBE_PATH" "https://storage.googleapis.com/kubernetes-release/release/$LATEST_KUBECTL/bin/$PLATFORM/amd64/kubectl"
        chmod +x "$KUBE_PATH"
    fi

    if [[ "$KUBECONFIG" == "" ]]; then
        export KUBECONFIG="$DIR/conf/kube/kubeconfig-shard1.yaml"
    fi
    
    kubectl config view

    echo "" 
}

function get_yaml_path {
    local YAML_PATH="$DIR/../conf/kube/${1}-web-test${2}.yaml"
    readlink -f "$YAML_PATH" 2>/dev/null || echo "$YAML_PATH"
}

function make_yaml {
    local TEST_ID="$1"
    local COUNTER="$2"
    local TENANT_ID="$3"
    local FILE_PATH="$4"
    if [ ! -f "$FILE_PATH" ]; then
        cat > "$FILE_PATH" <<YAML
apiVersion: v1
kind: Pod
metadata:
  name: "${TENANT_ID}-kube-web-test${COUNTER}"
  labels:
    app: web-ms-demo
    test_id: "${TEST_ID}"
  annotations:
    aaa: bbbb
    containers-label.alpha.kubernetes.io/com.swarm.tenant.0: sf7f413cb-a678-412d-b024-8e17e28bcb88-default
spec:
  containers:
    - name: kube-web-server
      image: mrsabath/web-ms:v3
      ports:
        - containerPort: 80
      env:
        -
         name: "TEST"
         value: "web-test${COUNTER}"
YAML
    fi
}

function delete_if_exists {
    local NAME="$1"
    echo "*** Deleting pod $NAME if exists ***"
    kubectl delete pod "$NAME"
    echo ""
}

# Run main

main "$@"
