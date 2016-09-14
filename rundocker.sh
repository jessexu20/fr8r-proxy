#!/bin/bash
#set -v

# Help menu
helpme()
{
    cat <<HELPMEHELPME

Syntax: ${0} <env_name> [<args>]
Where:
    env_name - name of the environment, e.g: dev-vbox
    args:
    -d - run api-proxy container in the background (optional)
    -l [INFO, WARNING, ERROR, FATAL] - set the log level for the proxy (optional)
    -v [non-negative integer] - set the log verbosity for the proxy (optional)
    -n - configuration for running with Nginx, no-SSL (optional)
    -i [image_name] - run local image (optional), instead of public image [containercafe/api-proxy] (default)

HELPMEHELPME
}

function main {
    if [[ "$1" == "" || "$1" == -* ]] ; then
        echo 'env_name must be set'
        helpme
        exit 1
    fi
    local env_name="$1"

    # manage certificates - copy them to admin-certs
    #local CERT_MASTER="../ansible/certs/dev-vbox-radiant01"
    local CERTS="$HOME/.openradiant/envs/$env_name"
    local ACERTS="$CERTS/admin-certs"
    local IMG="containercafe/api-proxy"
    if [ ! -d "$ACERTS" ]; then
      echo "missing $ACERTS directory. Execute ansible scripts first"
      exit 99
    fi

    # generte new api-proxy certs only if the hjproxy.cert is missing in $ACERTS
    # assuming all the certs are delete when new CA is created 
	if [ ! -e "$ACERTS/hjserver.pem" ]; then
		echo "ERROR: missing API Proxy server certs in $ACERTS"
		echo "       execute \"setupdocker.sh $env_name\" first"
		exit 98
	fi

	# check if creds.json was created
	if [ ! -e "$CERTS/creds.json" ]; then
		echo "ERROR: missing $CERTS/creds.json file"
		echo "       execute \"setupdocker.sh $env_name\" first"
		exit 97
	fi

    # to run container as a daemon use `-d` flag:
    local EXTRA_FLAGS=()
    shift
    while test $# -gt 0; do
        case "$1" in
            -l)
                EXTRA_FLAGS+=("-e" "log_level=$2")
                shift 2
                ;;
            -v)
                EXTRA_FLAGS+=("-e" "log_verbosity=$2")
                shift 2
                ;;
            -i)
                IMG=$2
                echo "Using $IMG image"
                shift 2
                ;;
            -n)
                EXTRA_FLAGS+=("-e" "use_api_key_header=true" "-e" "use_api_key_cert=false" "-e" "tls_inbound=false")
                shift
                ;;
            *)
                EXTRA_FLAGS+=("$1")
                shift
                ;;
        esac
    done

    # remove the previous container instance 
    docker ps -a | grep api-proxy &> /dev/null
    if [ $? == 0 ]; then
        echo "A previous api-proxy container exists -> removing it now..."
        docker rm -f api-proxy
    fi
    
    set -x
    # start new container instance using public api proxy image. Map the volume to CERTS
    #docker run "${EXTRA_FLAGS[@]}" -v "$CERTS":/opt/tls_certs -p 8087:8087 -e "env_name=$env_name" --name api-proxy containercafe/api-proxy
    # to run your own image, built using `builddocker.sh` script, comment out the line above on un-comment below:
    docker run "${EXTRA_FLAGS[@]}" -v "$CERTS":/opt/tls_certs -p 8087:8087 -e "env_name=$env_name" --name api-proxy $IMG
}

function copy_certs {
    local CERT_MASTER="$1"
    local ACERTS="$2"

    # copy the master certs
    cp -f "$CERT_MASTER/ca"* "$ACERTS"
    cp -f "$CERT_MASTER/admin-key.pem" "$ACERTS/kadmin.key"
    cp -f "$CERT_MASTER/admin.pem" "$ACERTS/kadmin.pem"

    cp -f ../ansible/roles/keygen-shard/files/api-proxy-openssl.cnf "$ACERTS/api-proxy-openssl.cnf"

    # generate all proxy certs
    ./gen_server_certs.sh "$ACERTS"
}

function verify_certs_matching {
    local CERT_MASTER="$1"
    local ACERTS="$2"

    local matching=true
    compare_certs "$CERT_MASTER/ca.pem" "$ACERTS/ca.pem" || matching=false
    compare_certs "$CERT_MASTER/admin.pem" "$ACERTS/kadmin.pem" || matching=false

    echo "$matching"
}

function compare_certs {
    local CERT1="$1"
    local CERT2="$2"
    
    if [[ -f "$CERT1" && -f "$CERT2" && `diff --brief <(cert_to_text "$CERT1") <(cert_to_text "$CERT2")` = "" ]]; then
        return 0
    else
        return 1
    fi
}

function cert_to_text {
    local CERT_PATH="$1"
    
    openssl x509 -in "$CERT_PATH" -text -noout 2>&1
}

# Main

main "$@"
