FROM golang:1.6.2

# Add Labels to image that shows git build details in metadata
ARG git_commit_id=unknown
ARG git_remote_url=unknown
ARG build_date=unknown
ARG jenkins_build_id=unknown
ARG jenkins_build_number=unknown

# Log levels are INFO, WARNING, ERROR, FATAL
ARG log_level=INFO

# Log verbosity is a non-negative integer
ARG log_verbosity=0

LABEL git-commit-id=${git_commit_id} \
      git-remote-url=${git_remote_url} \
      build-date=${build_date} \
      jenkins-build-id=${jenkins_build_id} \
      jenkins-build-number=${jenkins_build_number}

WORKDIR /api-proxy

ENV GOPATH /api-proxy
RUN apt-get update
RUN apt-get -y install --no-install-recommends ca-certificates curl git \
 && go get "github.com/golang/glog"

COPY src/ ./src/
RUN go build -o bin/api-proxy src/api-proxy/api-proxy.go

ENV log_file_path="/tmp/feeds/logstash_api_proxy" \
    docker_port="8089" \
    docker_api_ver="/v1.18" \
    tls_inbound="true" \
    tls_outbound="true" \
    use_api_key_header="false" \
    use_api_key_cert="true" \
    ca_cert_file="/opt/tls_certs/admin-certs/ca.pem" \
    client_cert_file="/opt/tls_certs/admin-certs/hjclient.pem" \
    client_key_file="/opt/tls_certs/admin-certs/hjclient.key" \
    server_cert_file="/opt/tls_certs/admin-certs/hjserver.pem" \
    server_key_file="/opt/tls_certs/admin-certs/hjserver.key" \
    kube_admin_key_file="/opt/tls_certs/admin-certs/admin-key.pem" \
    kube_admin_cert_file="/opt/tls_certs/admin-certs/admin.pem" \
    stub_auth_file="/opt/tls_certs/creds.json" \
    surrogate_ids="false" \
    max_container_conn="4" \
    max_node_conn="0" \
    kube_port="443" \
    swarm_node_port="2375" \
    registry_location="" \
    registry_admin_password= \
    consul_ip="" \
    consul_port="" \
    log_level=${log_level} \
    log_verbosity=${log_verbosity}

# copy all the scripts needed by proxy
COPY create_tenant.sh mk_user_cert.sh mk_kubeconfig.sh build.info ./

EXPOSE 6969
CMD ["/api-proxy/bin/api-proxy", "6969"]
