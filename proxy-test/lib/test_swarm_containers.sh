#!/bin/bash

source "$(dirname "$0")/common_test_func.sh"

function main {
    local LOGS_PATH="$1"
    local TENANT_ID="$2"
    local PARALLEL="$4"
    local NUM_CONTAINERS="$3"
    local TEST_ID="$( date +%Y%m%d )$( date +%H%M%S )"

    init_tests "$TENANT_ID" "swarm" "$LOGS_PATH"
    
    test_docker_commands "$TEST_ID"
    
    local COUNTER=1
    while [ $COUNTER -le $NUM_CONTAINERS ]; do

        if [[ $PARALLEL == true ]]; then
            test_non_existent_container "$TEST_ID" "$TENANT_ID" $COUNTER &
            increment_test_count
        else
            test_non_existent_container "$TEST_ID" "$TENANT_ID" $COUNTER
        fi

        let COUNTER++
    done
    
    COUNTER=1
    while [ $COUNTER -le $NUM_CONTAINERS ]; do

        if [[ $PARALLEL == true ]]; then
            test_creation_container_without_network "$TEST_ID" "$TENANT_ID" $COUNTER &
            increment_test_count
        else
            test_creation_container_without_network "$TEST_ID" "$TENANT_ID" $COUNTER
        fi

        let COUNTER++
    done

    COUNTER=1
    while [ $COUNTER -le $NUM_CONTAINERS ]; do
    
        #if [[ $PARALLEL == true ]]; then
        #    test_creation_container_with_network "$TEST_ID" "$TENANT_ID" $COUNTER &
        #    increment_test_count
        #else
        #    test_creation_container_with_network "$TEST_ID" "$TENANT_ID" $COUNTER
        #fi

        let COUNTER++
    done
    
    wait
    
    COUNTER=1
    while [ $COUNTER -le $NUM_CONTAINERS ]; do

        if [[ $PARALLEL == true ]]; then
            test_runtime_container_without_network "$TEST_ID" "$TENANT_ID" $COUNTER &
            increment_test_count
        else
            test_runtime_container_without_network "$TEST_ID" "$TENANT_ID" $COUNTER
        fi

        let COUNTER++
    done

    COUNTER=1
    while [ $COUNTER -le $NUM_CONTAINERS ]; do
    
        #if [[ $PARALLEL == true ]]; then
        #    test_runtime_container_with_network "$TEST_ID" "$TENANT_ID" $COUNTER &
        #    increment_test_count
        #else
        #    test_runtime_container_with_network "$TEST_ID" "$TENANT_ID" $COUNTER
        #fi

        let COUNTER++
    done
    
    wait

    COUNTER=1
    while [ $COUNTER -le $NUM_CONTAINERS ]; do

        if [[ $PARALLEL == true ]]; then
            test_deletion_container_without_network "$TEST_ID" "$TENANT_ID" $COUNTER &
            increment_test_count
        else
            test_deletion_container_without_network "$TEST_ID" "$TENANT_ID" $COUNTER
        fi
    
        let COUNTER++
    done
    
    COUNTER=1
    while [ $COUNTER -le $NUM_CONTAINERS ]; do

        #if [[ $PARALLEL == true ]]; then
        #    test_deletion_container_with_network "$TEST_ID" "$TENANT_ID" $COUNTER &
        #    increment_test_count
        #else
        #    test_deletion_container_with_network "$TEST_ID" "$TENANT_ID" $COUNTER
        #fi
    
        let COUNTER++
    done
    
    wait

    # Make sure everything is clean at the end
    assert_not_running "${TENANT_ID}"

    complete_tests "Docker Containers"
}

# Tests

function test_docker_commands {
    local TEST_ID="$1"

    begin_test_block

    #assert_images 0
    #assert_images_a 0
    
    assert_ps 0
    assert_ps_a 0
    
    assert_network_ls 0
    assert_network_inspect "none" 1
    #assert_network_inspect "default" 0

    end_test_block
}

function test_non_existent_container {
    local TEST_ID="$1"
    local TENANT_ID="$2"
    local COUNTER="$3"
    local container_name="${TENANT_ID}_nonexistent_test${COUNTER}"
    
    delete_if_exists "$container_name"
    
    begin_test_block
    
    assert_not_running "$container_name" "$TEST_ID"
    
    assert_inspect "$container_name" 1
    assert_start "$container_name" 1
    assert_stop "$container_name" 1
    
    assert_never_ran "$container_name" "$TEST_ID"
    
    end_test_block
}

function test_creation_container_without_network {
    _test_creation_container "$@" "nonet" --net none
}

function test_creation_container_with_network {
    _test_creation_container "$@" "net"
}

function _test_creation_container {
    local TEST_ID="$1"
    local TENANT_ID="$2"
    local COUNTER="$3"
    local CONTAINER_ID="$4"
    local container_name="${TENANT_ID}_${CONTAINER_ID}_test${COUNTER}"
    shift 4
    
    delete_if_exists "$container_name"

    begin_test_block

    assert_not_running "$container_name" "$TEST_ID"
    
    assert_run "$container_name" "$TEST_ID" 0 "$@"
    assert_running "$container_name" "$TEST_ID"

    assert_inspect "$container_name" 0

    assert_stop "$container_name" 0
    assert_not_running "$container_name" "$TEST_ID"
    assert_ps_a 0

    assert_start "$container_name" 0
    assert_running "$container_name" "$TEST_ID"
    
    end_test_block
}

function test_runtime_container_without_network {
    _test_runtime_container "$@" "nonet"
}

function test_runtime_container_with_network {
    _test_runtime_container "$@" "net"
}

function _test_runtime_container {
    local TEST_ID="$1"
    local TENANT_ID="$2"
    local COUNTER="$3"
    local CONTAINER_ID="$4"
    local container_name="${TENANT_ID}_${CONTAINER_ID}_test${COUNTER}"
    
    begin_test_block

    # Test can't remove without stopping first
    assert_rm "$container_name" 1
    
    assert_logs "$container_name" 0
    
    end_test_block
}

function test_deletion_container_without_network {
    _test_deletion_container "$@" "nonet"
}

function test_deletion_container_with_network {
    _test_deletion_container "$@" "net"
}

function _test_deletion_container {
    local TEST_ID="$1"
    local TENANT_ID="$2"
    local COUNTER="$3"
    local CONTAINER_ID="$4"
    local container_name="${TENANT_ID}_${CONTAINER_ID}_test${COUNTER}"

    if [ $(( $COUNTER % 2 )) -eq 0 ]; then
        begin_test_block
    
        assert_stop "$container_name" 0
        assert_rm "$container_name" 0
        
        end_test_block
    else
        assert_rm_f "$container_name" 0
    fi
}

function assert_ps {
    assert docker ps \
           --equal $1 \
           --log "docker ps"
}

function assert_ps_a {
    assert docker ps -a \
           --equal $1 \
           --log "docker ps -a"
}

function assert_not_running {
    assert docker ps -f status=running -f label=test_id="$2" \
           --equal 0 \
           --output-not-contains "$1" \
           --log "docker ps"
}

function assert_running {
    assert docker ps -f status=running -f label=test_id="$2" \
           --equal 0 \
           --output-contains "$1" \
           --log "docker ps"
}

function assert_never_ran {
    assert docker ps -f label=test_id="$2" \
           --equal 0 \
           --output-not-contains "$1" \
           --log "docker ps -a"
}

function assert_run {
    assert docker run -d --name "$1" --memory 128m --label test_id="$2" "${@:4}" mrsabath/web-ms \
           --equal $3 \
           --log "docker run; name = $1"
}

function assert_inspect {
    assert docker inspect "$1" \
           --equal $2 \
           --log "docker inspect; name = $1"
}

function assert_stop {
    assert docker stop "$1" \
           --equal $2 \
           --log "docker stop; name = $1"
}

function assert_start {
    assert docker start "$1" \
           --equal $2 \
           --log "docker start; name = $1"
}

function assert_rm {
    assert docker rm "$1" \
           --equal $2 \
           --log "docker rm; name = $1"
}

function assert_rm_f {
    assert docker rm -f "$1" \
           --equal $2 \
           --log "docker rm -f; name = $1"
}

function assert_logs {
    assert docker logs --tail 1 "$1" \
           --equal $2 \
           --log "docker logs; name = $1"
}

function assert_network_ls {
    assert docker network ls \
           --equal $1 \
           --log "docker network ls"
}

function assert_network_inspect {
    assert docker network inspect "$1" \
           --equal $2 \
           --log "docker network inspect; name = $1"
}

function assert_images {
    assert docker images \
           --equal $1 \
           --log "docker images"
}

function assert_images_a {
    assert docker images -a \
           --equal $1 \
           --log "docker images -a"
}

# Utils

function delete_if_exists {
    local NAME="$1"
    echo "*** Deleting container $NAME if exists ***"
    docker rm -f "$NAME"
    echo ""
}

# Run main

main "$@"
