#!/bin/bash

# this script is invoked as ./mk_user_cert.sh "$API_KEY" "$SHARD_NAME"

# Required parameters
CA_PASSWORD="thesecret"
API_KEY="$1"
SHARD_NAME="$2"

if [ "$CA_ROOT" == "" ] ; then
    CA_ROOT="/opt/tls_certs/admin-certs"
fi

if [ "$CERT_ROOT" == "" ] ; then
    USER_ROOT="/opt/tls_certs/$SHARD_NAME/$API_KEY"
else
    USER_ROOT="$CERT_ROOT/$API_KEY"
fi

# Store the certificates in provided directory
if [ -d "$USER_ROOT" ]; then
    echo "Certificates already exist for this Apikey"
    exit 1
fi
mkdir -p "$USER_ROOT"
cd "$USER_ROOT"


# Generate a unique user cert and output to stdout
openssl genrsa -out key.pem 2048
if [ $? -ne 0 ]; then
    echo "Certificates could not be created"
    exit 1
fi

openssl req -subj "/CN=$API_KEY" -new -key key.pem -out client.csr
if [ $? -ne 0 ]; then
    echo "Certificates could not be created"
    exit 1
fi

echo extendedKeyUsage = clientAuth > extfile.cnf
openssl x509 -passin "pass:${CA_PASSWORD}" -req -days 3650 -in client.csr -CA "${CA_ROOT}/ca.pem" -CAkey "${CA_ROOT}/ca-key.pem" -CAcreateserial -out cert.pem -extfile extfile.cnf
if [ $? -ne 0 ]; then
    echo "Certificates could not be created"
    exit 1
fi


# Copy ca.pem into directory
cp "${CA_ROOT}/ca.pem" ca.pem


echo `pwd` # To pass back to create_tenant.sh

# Remove client.csr & extfile.cnf
rm client.csr extfile.cnf
