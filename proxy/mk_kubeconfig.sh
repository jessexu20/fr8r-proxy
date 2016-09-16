#!/bin/bash

function make_kubeconfig {
    local CONFIG_DIR="$1"
    local TENANT_ID="$2"
    local PROXY_IP="$3"

    cat > "$CONFIG_DIR/kube-config" <<YAML
apiVersion: v1
clusters:
- cluster:
    server: https://${PROXY_IP}:6969
    certificate-authority: ca.pem
  name: shard1
contexts:
- context:
    cluster: shard1
    namespace: s${TENANT_ID}-default
    user: $TENANT_ID
  name: shard1
current-context: shard1
kind: Config
preferences: {}
users:
- name: ${TENANT_ID}
  user:
    client-certificate: cert.pem
    client-key: key.pem
YAML
}

make_kubeconfig "$@"
