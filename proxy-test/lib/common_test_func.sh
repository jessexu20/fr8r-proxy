#!/bin/bash

# init_tests - Initialize a test session
function init_tests {
    TENANT_ID="$1"
    TEST_TYPE="$2"
    RESULTS_PATH="$3"
    TEST_COUNT="1"
    TEST_START_TIME=$(date +%s)

    create_log_file "$RESULTS_PATH"
}

# complete_tests - End a test session and summarize the results
function complete_tests {
    local TEST_TYPE="$1"
    local LOG_FILE="$RESULTS_PATH"
    local elapsed_time=$(($(date +%s) - TEST_START_TIME))

    local FAIL_COUNT=`grep -cE "^[0-9.]+,FAIL" "$LOG_FILE"`
    local PASS_COUNT=`grep -cE "^[0-9.]+,PASS" "$LOG_FILE"`
    local TOTAL_COUNT=$((FAIL_COUNT + PASS_COUNT))

    printf '\n%*s\n\n' "${COLUMNS:-$(tput cols)}" '' | tr ' ' - >> "$LOG_FILE"
    
    cat >> "$LOG_FILE" <<SUMMARY
$TEST_TYPE Test summary:
Total = $TOTAL_COUNT tests
Passed = $PASS_COUNT tests
Failed = $FAIL_COUNT tests
Total time = $elapsed_time sec
SUMMARY
}

# create_log_file - Create the log file for a test session and print the env
function create_log_file {
    local LOG_FILE="$1"
    
    mkdir -p "$(dirname "$LOG_FILE")"

    rm -f "$LOG_FILE"
    
    cat > "$LOG_FILE" <<ENV
DOCKER_HOST=$DOCKER_HOST
DOCKER_TLS_VERIFY=$DOCKER_TLS_VERIFY
DOCKER_CERT_PATH=$DOCKER_CERT_PATH
KUBECONFIG=$KUBECONFIG

ENV
}

# increment_counter - Increment a counter
function increment_counter {
    local last=`echo "$1" | rev | cut -d. -f1 | rev`
    let last++
    sed -r "s/[0-9]+$/$last/" <<< "$1"
}

# increment_test_counter - Increment the test count
function increment_test_count {
    TEST_COUNT=`increment_counter $TEST_COUNT`
}

# begin_test_block - Mark the start of a test block
function begin_test_block {
    TEST_COUNT="${TEST_COUNT}.1"
}

# end_test_block - Mark the start of a test block
function end_test_block {
    TEST_COUNT=`increment_counter "$(echo "$TEST_COUNT" | rev | cut -d. -f2- | rev)"`
}

# assert - Assert
function assert {
    local TEST_CMD=()
    local EQUALS=""
    local OUTPUT_CONTAINS=""
    local OUTPUT_NOT_CONTAINS=""
    local TEST_NAME=

    local test_count="$TEST_COUNT"
    increment_test_count
    
    local expected=""
    
    while test $# -gt 0; do
        case "$1" in 
            "--equal")
                EQUALS="$2"
                expected="$expected and exit code equals $2"
                shift 2
                ;;
            "--output-contains")
                OUTPUT_CONTAINS="$2"
                expected="$expected and output contains '$2'"
                shift 2
                ;;
            "--output-not-contains")
                OUTPUT_NOT_CONTAINS="$2"
                expected="$expected and output does not contains '$2'"
                shift 2
                ;;
            "--log")
                TEST_NAME="$2"
                shift 2
                ;;
            *)
                TEST_CMD+=("$1")
                shift
                ;;
        esac
    done
    
    if [ "$expected" = "" ]; then
        echo "Missing test condition for ${TEST_CMD[@]}"
        echo "Missing test condition for ${TEST_CMD[@]}" >> "$RESULTS_PATH"
    else
        expected=`cut -c6- <<<"$expected"`
    fi

    echo -e "Running test $test_count *** $TEST_NAME ***\n" 

    local test_completed=false
    until [ "$test_completed" = true ]; do
        local output=""
        local timestamp=$(date +"%Y%m%d.%H%M%S")
        local start_time=$(date +%s)
        output=$("${TEST_CMD[@]}" 2>&1)
        local RESULT=$?
        local end_time=$(date +%s)
        
        if [ $(grep -c "docker: Error response from daemon: Task launched with invalid offers: Offer " <<<"$output") -eq 0 ]; then
            test_completed=true
        fi
    done

    echo -e "Test $TEST_NAME completed with code $RESULT ($EXPECTED expected):\n$output\n"

    local log_command=$(printf " %s" "${TEST_CMD[@]}")
    log_command=${log_command:1}

    local success=true
    _test_equals "$RESULT" "$EQUALS" || success=false
    _test_contains "$output" "$OUTPUT_CONTAINS" || success=false
    _test_not_contains "$output" "$OUTPUT_NOT_CONTAINS" || success=false

    if [ "$success" = true ]; then
        output="OK"
    else
        output=`tr "\n" " " <<< "$output"`
    fi

    log_test_result "$timestamp" "$success" $((end_time - start_time)) "$TENANT_ID" "$TEST_TYPE" "$test_count" "$log_command" "$RESULT" "$expected" "$output" "$RESULTS_PATH"
}

# _test_equals - Internal, test equality between numbers
function _test_equals {
    local RESULT="$1"
    local EXPECTED="$2"
    
    if [ "$EXPECTED" = "" ]; then
        return 0
    else
        [ "$RESULT" -eq "$EXPECTED" ]
    fi
}

# _test_contains - Internal, test if a string contains another string
function _test_contains {
    local INPUT="$1"
    local STRING="$2"

    if [ "$STRING" = "" ]; then
        return 0
    else
        [ $(grep -cE "$STRING" <<<"$INPUT") -gt 0 ]
    fi
}

# _test_not_contains - Internal, test if a string doesn't contains another string
function _test_not_contains {
    local INPUT="$1"
    local STRING="$2"

    if [ "$STRING" = "" ]; then
        return 0
    else
        [ $(grep -cE "$STRING" <<<"$INPUT") -eq 0 ]
    fi
}

# log_test_result - Append the test result to the log file
function log_test_result {
    local TIMESTAMP="${1}"
    local SUCCESS="${2}"
    local TIME="${3}"
    local TENANT_ID="${4}"
    local TYPE="${5}"
    local TEST_COUNT="${6}"
    local SUMMARY="${7}"
    local RESULT="${8}"
    local EXPECTED="${9}"
    local CMD_OUTPUT="${10}"
    local LOG_FILE="${11}"

    if [[ "$SUCCESS" == true ]]; then
        SUCCESS="PASS"
    else
        SUCCESS="FAIL"
    fi

    echo -e "$TIMESTAMP,$SUCCESS,${TIME}sec,$TENANT_ID,$TYPE,Test $TEST_COUNT,$SUMMARY,$RESULT,$EXPECTED,$CMD_OUTPUT\n" >> "$LOG_FILE"
}
