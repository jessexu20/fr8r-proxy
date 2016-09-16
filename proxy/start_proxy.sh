#!/bin/bash

# override TLS_OUTBOUND to support K8s
export tls_outbound=true
export tls_inbound=true
export server_key_file="admin-certs/hjserver.key"
export server_cert_file="admin-certs/hjserver.pem"
export ca_cert_file="admin-certs/ca.pem"
export stub_auth_file="creds.json"
export kube_admin_key_file="admin-certs/kadmin.key"
export kube_admin_cert_file="admin-certs/kadmin.pem"

# dev-mon01:
export ccsapi_host=10.140.34.174:8081
export consul_ip=10.140.34.174

# copy the certs
cp -f ../ansible/certs/dev-vbox-shard1/ca* "$(dirname "$ca_cert_file")"
cp -f ../ansible/certs/dev-vbox-shard1/admin-key.pem "$kube_admin_key_file"
cp -f ../ansible/certs/dev-vbox-shard1/admin.pem "$kube_admin_cert_file"
./gen_server_certs.sh

# start proxy
bin/api-proxy 
