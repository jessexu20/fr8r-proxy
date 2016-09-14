FROM ubuntu:xenial

# kubectl version
ARG k8s_version=1.3.5

# docker version
ARG docker_version=1.12.0

# test tenants environment
ARG env_name

# set the environment
ENV ENV_NAME=${env_name} \
    LOGS_DIR=/tests/logs \
    CERTS_DIR=/tests/certs

# Install curl, kubectl and docker
RUN apt-get -qq update \
 && DEBIAN_FRONTEND=noninteractive apt-get -qq -y install curl \
 && curl -fsSLo /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v${k8s_version}/bin/linux/amd64/kubectl \
 && chmod +x /usr/local/bin/kubectl \
 && curl -fsSLO https://get.docker.com/builds/Linux/x86_64/docker-${docker_version}.tgz \
 && tar --strip-components=1 -xzf docker-${docker_version}.tgz -C /usr/local/bin \
 && rm -f docker-${docker_version}.tgz

# copy the scripts
WORKDIR /tests
COPY conf conf
COPY lib lib
COPY test_containers.sh docker/test_entrypoint.sh ./

ENTRYPOINT [ "/tests/test_entrypoint.sh" ]
