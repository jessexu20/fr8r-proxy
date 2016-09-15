#!/bin/bash

export PASSWD=thesecret

if [[ "$1" == "" ]] ; then
    echo "admin certs dir must be provided"
    exit 1
fi

cd "$1"
openssl genrsa -out hjserver.key 4096
openssl req -new -key hjserver.key -out hjserver.csr -subj "/CN=localhost" -config api-proxy-openssl.cnf
openssl x509 -req -in hjserver.csr -CA ca.pem -CAkey ca-key.pem -CAserial ca.srl -days 1500 -extensions v3_req -out hjserver.pem -passin pass:$PASSWD -extfile api-proxy-openssl.cnf
