#!/bin/bash

# Help menu
helpme()
{
    cat <<HELPMEHELPME

Syntax: ${0} <tenant_id> <shard_name> <shard_ip> <api_proxy_ip>
Where:
    tenant_id = id of the desired tenant
    shard_name = name of the shard, e.g: shard1
    shard_ip = IP address of the shard, e.g: 192.168.10.2
    api_proxy_ip = IP address of API Proxy, e.g: 192.168.10.4

 Note: $env_name must be defined via environment variable

HELPMEHELPME
}

function generate_api_key {
    local key_len=$1
    cat /dev/urandom | env LC_CTYPE=C tr -dc 'a-zA-Z0-9' | fold -w $key_len | head -n 1
}

function append_to_creds {
    local creds_path="$1"

    if [ ! -f "$creds_path" ]; then
        echo "Creating $(basename "$creds_path") file"
        touch "$creds_path"
    fi

    echo "Writing to $(basename "$creds_path")"
    echo "{\"Status\":$STATUS, \"Node\":\"$SHARD_IP\", \"Docker_id\":\"$DOCKER_ID\", \"Container\":\"$CONTAINER\", \"Swarm_shard\":$SWARM_SHARD, \"Tls_override\":$TLS_OVERRIDE, \"Space_id\":\"$SPACE_ID\", \"Reg_namespace\":\"$REG_NAMESPACE\", \"Apikey\":\"$API_KEY\", \"Orguuid\":\"$ORGUUID\", \"Userid\":\"$USERID\", \"Endpoint_type\":\"$ENDPOINT_TYPE\", \"TLS_path\":\"$TLS_dir\"}" >> "$creds_path"
}

# Check args
if [[ "$1" == "-?" || "$1" == "-h" || "$1" == "--help" || "$1" == "help" ]] ; then
    helpme
    exit 1
elif [[ "$4" == "" ]] ; then
    echo "Incorrect number of arguments"
    helpme
    exit 1
elif [[ "$env_name" == "" ]] ; then
    echo '$env_name must be set as environment variable'
    helpme
    exit 1
fi

# Variables
API_KEY_LEN=48
SPACE_ID="$1"
STATUS=200

SHARD_IP="$3"
SHARD_NAME="$2"
PROXY_IP="$4"

DOCKER_ID=""
CONTAINER=""
SWARM_SHARD=true
TLS_OVERRIDE=true
REG_NAMESPACE="swarm"
ORGUUID="orgname"
USERID="userid"
ENDPOINT_TYPE="radiant"

# Generate API key
echo "Generating API key of length $API_KEY_LEN"
API_KEY=`generate_api_key $API_KEY_LEN`
echo "Generated API key: $API_KEY"

# Create certificate
# WHAT ABOUT LOCATION OF CA / PASSWORD rn just my configuration
echo "Creating certificates"
TLS_dir=$(./mk_user_cert.sh "$API_KEY" "$SHARD_NAME")

# Right now, doing only 1 space per user
if [ $? -eq 1 ]; then
    echo "$TLS_dir"
    exit 1
fi

# Append the user to the creds.json file
append_to_creds "$stub_auth_file"

# Setup the environment
DOCKER_HOST="$PROXY_IP:6969"
DOCKER_TLS_VERIFY=1
if [ "$CERT_ROOT" = "" ] ; then
    # when running from the container, specify the path to shared volume
    DOCKER_CERT_PATH="~/.fr8r/envs/$env_name/$SHARD_NAME/$API_KEY"
else
    # when running as a script, use actual path
    DOCKER_CERT_PATH="$TLS_dir"
fi

cat <<ENV
Certificates created for Apikey $API_KEY
Located at $TLS_dir

# Setup docker environment:
export DOCKER_HOST=$DOCKER_HOST
export DOCKER_TLS_VERIFY=$DOCKER_TLS_VERIFY
export DOCKER_CERT_PATH=$DOCKER_CERT_PATH

ENV

# Make kube-config
./mk_kubeconfig.sh "$TLS_dir" "$SPACE_ID" "$PROXY_IP"
if [ $? -eq 1 ]; then
    echo "Unable to create kube-config file. Terminating."
    exit 1
fi

cat <<ENV
# Setup kubernetes environment:
export KUBECONFIG=$DOCKER_CERT_PATH/kube-config

ENV

# Initilize the kubernetes tenant
RC=`curl -w "%{http_code}" -k -XPOST -s -o /dev/null --cert "$TLS_dir/cert.pem" --key "$TLS_dir/key.pem" --cacert "$TLS_dir/ca.pem" -H "Content-Type: application/json"  https://localhost:6969/kubeinit  2>&1`
if [ "$RC" != "200" ] ; then
   echo "ERROR: Could not initialize kubernetes tenant"
fi

cd $TLS_dir
tar -zcvf $API_KEY.tgz *
cd -
