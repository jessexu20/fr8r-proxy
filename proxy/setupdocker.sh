#!/bin/bash
#set -v

# Help menu
helpme()
{
    cat <<HELPMEHELPME

Syntax: ${0} <env_name> 
Where:
    env_name - name of the environment, e.g: dev-vbox

HELPMEHELPME
}

if [[ "$1" == "" || "$1" == -* ]] ; then
    echo 'env_name must be set'
    helpme
    exit 1
fi
env_name="$1"

CERTS="$HOME/.fr8r/envs/$env_name"
ACERTS="$CERTS/admin-certs"

	if [ ! -d "$ACERTS" ]; then
	echo "missing $ACERTS directory. Execute ansible scripts first"
	exit 99
fi

# generate new api-proxy certs only if the hjproxy.cert is missing in $ACERTS
# assuming all the certs are delete when new CA is created 
if [ ! -e "$ACERTS/hjserver.pem" ]; then
	cp -f ../ansible/roles/keygen-shard/files/api-proxy-openssl.cnf "$ACERTS/api-proxy-openssl.cnf"
	# generate all proxy certs
	./gen_server_certs.sh "$ACERTS"
else
	echo "WARNING: using the existing api-proxy cert in $ACERTS"
	echo "To recreate the api-proxy certs, delete $ACERTS/hjserver.pem"
fi

if [ ! -e "$CERTS/creds.json" ]; then
	touch "$CERTS/creds.json"
else
	echo "WARNING: using the existing $CERTS/creds.json file"
	echo "To recreate it, delete first $CERTS/creds.json and re-run this script"
fi
