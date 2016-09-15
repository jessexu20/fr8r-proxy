# OpenRadiant Proxy
Proxy is the component of OpenRadiant that intercepts the communication between
the clients (Docker or Kubernetes) and the OpenRadiant cluster, using HTTP session
hijacking. It validates the tenant and provided TLS certificates.
It also redirects to proper shard when cluster sharing is used.

More detailed documentation about proxy is available [here](../docs/proxy.md)

## Proxy Environment Setup and Development
Steps below explain the basic configuration and setup to run Proxy. This setup
assumes the target system already has Docker installed, the Proxy will be deployed
locally to manage Swarm and Kubernetes instances also installed locally, using
the ansible scripts described in [Quick Start](../README.md#quick-start)
For other deployments, one can still follow these steps, with adjustments provided below.

The steps below work best with native docker [for Mac, Unix or Windows](http://www.docker.com/products/overview)
It would also work with `docker-machine`, but it requires additional steps. Look
for /[DOCKER MACHINE] tag.

### Step 1: Get proxy code and run it
[*This will be done by ansible install script*]
If you have not done this already, clone the repository:

```bash
git clone git@github.com:containercafe/containercafe.git
# or
git clone https://github.com/containercafe/containercafe.git
```

#### Build proxy image
The latest working image of API Proxy is already built and publicly available
[https://hub.docker.com/r/containercafe/api-proxy/](https://hub.docker.com/r/containercafe/api-proxy/)
To build the image locally, using your local code, execute:

```bash
cd proxy
./builddocker.sh
```
To publish the image to Docker Hub see the steps [here](https://github.com/containercafe/containercafe/blob/master/docs/building-images.md#containercafeproxy)

The proxy image is very small but it's slow to build.
If you are developing on the proxy you can build a bigger image in less time, execute:
```bash
cd proxy
./builddocker.sh -f Dockerfile.dev
```

#### Run proxy as a container
[*This will be done by ansible install script*]
Proxy service will be installed as a container on your default docker host.
You can either start the proxy based on the public image or using image based
on your own code (see above)

When starting the proxy, provide the environment name e.g. _dev-vbox_:

```bash
cd openradiant/proxy
./rundocker.sh <env_name>
```

FOR PROXY DEVELOPERS:

to run the local image of the proxy, use -i <image_name> flag:

```bash
./rundocker.sh <env_name> -i api-proxy
```

there is also an option available to start the api proxy in non-secure mode, useful
for interaction with Nginx. It will run without TLS input and allows passing
the API key in a header of a request instead of certificate.
The header name is "X-Tls-Client-Dn". See the manual example below.

A complete syntax rundocker.sh is here:

```bash
Syntax: rundocker.sh <env_name> [<args>]
Where:
    env_name - name of the environment, e.g: dev-vbox
    args:
    -d - run api-proxy container in the background (optional)
    -l [INFO, WARNING, ERROR, FATAL] - set the log level for the proxy (optional)
    -v [non-negative integer] - set the log verbosity for the proxy (optional)
    -n - configuration for running with Nginx, no-SSL (optional)
    -i [image_name] - run local image (optional), instead of public image [containercafe/api-proxy] (default)
```


This will start the Proxy as a container named `api-proxy`, running in the current
terminal, on port specified in [Dockerfile](Dockerfile) e.g:
```
EXPOSE 6969
CMD ["/api-proxy/bin/api-proxy", "6969"]
```

The Proxy container mounts the volume to the following local location:
```
~/.openradiant/envs/<env_name/
e.g:
~/.openradiant/envs/dev-vbox/
```
And they are mounted locally to `/opt/tls_certs/` from the container.

If you want to run the Proxy as background container (daemon), use the `-d` flag:
```
./rundocker.sh <env_name> -d
```

The Proxy container can be seen there along with its logs:
```
docker ps
docker logs -f api-proxy
```

To set a custom log level and/or log verbosity use the `-l` and `-v` flags:
```
./rundocker.sh <env_name> -l <INFO,WARNING,ERROR,FATAL> -v <non-negative integer>
```

#### Run proxy as a script
If you don't want to run the Proxy as a container, you can run directly as GoLang
application. This requires Go libraries installed, proxy code compiled and added
to go path.
Review [Dockerfile](Dockerfile) for more details.
Setup the environment and start the application:
```bash
source ./set_local_env.sh
./start_proxy.sh
```
NOTE: There is a problem when running the proxy as a script on mac. (See the
issue [#10](https://github.com/containercafe/containercafe/issues/10) Mac implements
their own native SSL libraries for curl, therefore passing certs that are not
in the keychain is a bit problematic. Install curl via Homebrew:
`brew install curl`, keep the native curl, update your PATH to point at the new
curl executable location. Now you should be able to run the scripts that invoke
curl commands with certs.



### Step 2: Creating a tenant (account administrator)
When proxy runs as a container on the same host, just open a new terminal and
follow the steps below. When proxy runs on a different docker host, ssh to this
host first:

```bash
cd examples/vagrant
vagrant ssh proxy
```

Go the proxy directory and then execute the script to create TLS certificates
and API key for the given tenant. The example is using tenant `test1`.
This script also requires the name and the IP of the shard and proxy
E.g.
```
docker ps
docker exec api-proxy /api-proxy/create_tenant.sh <tenant> <shard_name> <shared_ip> <api_proxy_ip>
docker exec api-proxy /api-proxy/create_tenant.sh test1 radiant01 192.168.10.2 192.168.10.4
```

This command will display the details about the newly created TLS certs, including
their location. Certificates are created on the mounted volumes, therefore they
are available inside the proxy container and for the current terminal.

You can create as many tenants as you like.

At the bottom of the output the script displays the docker environment setup for
the newly created tenant, including the location of the certs. Here is the sample
output:
```bash
# Setup docker environment:
export DOCKER_HOST=localhost:6969
export DOCKER_TLS_VERIFY=1
export DOCKER_CERT_PATH=~/.openradiant/envs/dev-vbox/radiant01/ITNqyoU6Xe6ttgq7yQNwOeaQm6Ms8vauJqEQclghh3sdzDpg

# Setup kubernetes environment:
export KUBECONFIG=~/.openradiant/envs/dev-vbox/radiant01/ITNqyoU6Xe6ttgq7yQNwOeaQm6Ms8vauJqEQclghh3sdzDpg/kube-config
```
Copy and paste these commands in *a new terminal* (otherwise will no be able to
  make anymore calls to proxy container).

Now you should be able to execute commands using OpenRadiant tenant that you
just created:

```
docker ps
# create a new container:
docker run -d --name test --net none -m 128m mrsabath/web-ms
docker inspect test

kubectl get pods
# create a new pod
# assuming you are in openradiant/proxy directory:
kubectl create -f ../examples/apps/k8s/pod-web.yaml

# create a new deployment:
kubectl run k1 --image=busybox sleep 864000
kubectl get deployment

# now you should be able to list all your Kubernetes containers using docker command:
docker ps

# you can also try ReplicationController, ReplicaSet and Deployments:
kubectl create -f ../examples/apps/k8s/nginx-rc.yaml
kubectl create -f ../examples/apps/k8s/frontend-rs.yaml
kubectl run test-run --image=mrsabath/web-ms:v3

kubectl get rc
kubectl get rc
kubectl get deployments

```
To run the proxy against a different OpenRadiant shard, pass the IP of this shard
as additional parameter of the script `create_tenant.sh`. E.g:
```
docker exec api-proxy /api-proxy/create_tenant.sh test2 radiant02 192.168.10.11 192.168.10.4
```

You can also manually change the values in "/api-proxy/creds.json" file that lists
all the valid tenants.  
To modify the value in the running container:
```
docker exec -it api-proxy /bin/bash
```

### To view the content of the current authorization file:

```bash
docker exec api-proxy cat /opt/tls_certs/creds.json
```

Every entry of the `creds.json` has this format:
```json
{"Status":200, "Node":"192.168.10.2", "Docker_id":"", "Container":"", "Swarm_shard":true, "Tls_override":true, "Space_id":"sample_entry", "Reg_namespace":"swarm", "Apikey":"PV9S5hQARFmg0pVJwaPxbP588GdVKeYF1YGOePDvRNAGpyl4", "Orguuid":"orgname", "Userid":"userid", "Endpoint_type":"radiant", "TLS_path":"/opt/tls_certs/radiant01/PV9S5hQARFmg0pVJwaPxbP588GdVKeYF1YGOePDvRNAGpyl4"}
```


**NOTE**: All default config options are defined in the [Dockerfile](Dockerfile),
and can be overridden using the docker -e option on [startup](rundocker.sh)

## Running request manually, without the kubectl or docker clients
API Proxy can be started with non-SSL, using -n flag (see above). Call must include
the Apikey value in the header. Apikey is the random string in creds.json file
for your tenant. E.g:

```bash
export APIKEY=PV9S5hQARFmg0pVJwaPxbP588GdVKeYF1YGOePDvRNAGpyl4
export PROXY=192.168.10.4:6969
curl -XGET -H "X-Tls-Client-Dn: /CN=$APIKEY" -H "Content-Type: application/json" $PROXY/api
```

## Running Test Scripts
There are 2 type of tests:

* test container for proxy running as a container [proxy-test/docker/README.md](proxy-test/docker/README.md)

* test scripts for proxy running as a container [proxy-test/README.md](proxy-test/README.md)


## Hints and Troubleshooting Errors

 * `Error response from daemon: client is newer than server (client API version: 1.24, server API version: 1.22)`
 run `export DOCKER_API_VERSION=1.22` before running any docker commands.

 * `An error occurred trying to connect: Get https://localhost:6969/v1.21/images/4a419cdeaf69/json: tls: oversized record received with length 20527[]`
 In order to fix this problem, use `DOCKER_TLS_VERIFY=""` prefix for running 'docker' command

 * `docker: Error response from daemon: Task launched with invalid offers: Offer ea1a4d71-cf69-4292-90e7-530c77a5458b-O1 is no longer valid.`
 There is a caching problem on Mesos. Issue [#33](https://github.com/containercafe/containercafe/issues/33)
 is tracking it. Simply just repeat your last command. It should purge the cache
 and work again.

 * `docker: Error response from daemon: driver failed programming external connectivity on endpoint hjproxy (0910f89f1b27f3b05081a0bcec3ceadb6d335873d191b3f055ff82257cf77e5d): Error starting userland proxy: write /port/tcp:0.0.0.0:6969:tcp:172.17.0.2:6969/ctl: errno 526.` Please make sure no
 other process is running on port specified for proxy. Standalone proxy test on 6969?

 * `Could not read CA certificate "~/.openradiant/<env>/<shard>/fprVv76aAWfrmxboOxsO6dbzfZcITidkIwBslPgMAchFfwZI/ca.pem": open ~/.openradiant/<env>/<shard>/fprVv76aAWfrmxboOxsO6dbzfZcITidkIwBslPgMAchFfwZI/ca.pem: no such file or directory`
 Are you sure you are running your docker commands from `openradiant/proxy/`
 directory?
