# Testing, for Proxy running as a Container
For running the tests as a container see [the docker folder](docker/README.md).  
Steps below explain configuration & setup to run Swarm & Kubernetes testing, while Proxy is running as a container.
Note: Steps below detail multi-user testing. Testing can also be done with a single-user.


## Step 1: Terminals
Open as many terminals as test tenants desired. In this example, we will have 3 tenants - test1, test2, test3.
Therefore, open 3 different terminal windows and cd to location where you have cloned this fr8r project.
Then cd to proxy directory. E.g:
```bash
cd ~/workspace/fr8r/proxy
```

## Step 2: Certificates for test tenants
In every terminal, execute the script to create one test tenant:
```bash
docker exec api-proxy /api-proxy/create_tenant.sh test1 shard1 192.168.10.2
docker exec api-proxy /api-proxy/create_tenant.sh test2 shard1 192.168.10.2
docker exec api-proxy /api-proxy/create_tenant.sh test3 shard1 192.168.10.2
```

This will create the certificates for this tenant and the configurations necessary to run the tests.
To view all the accounts valid for this proxy:
```bash
docker exec api-proxy cat /opt/tls_certs/creds.json
```

## Step 3: Environment Variables
The certificate creation script will output few export statements. For example: <br />
```bash
# Setup docker environment:
export DOCKER_HOST=localhost:6969
export DOCKER_TLS_VERIFY=1
export DOCKER_CERT_PATH=~/.fr8r/envs/dev-vbox/shard1/7uJNzJqK5T33A4j9XkH6Fd1dQwCza0zHGHeFokmRJOWfz87I

# Setup kubernetes environment:
export KUBECONFIG=~/.fr8r/envs/dev-vbox/shard1/7uJNzJqK5T33A4j9XkH6Fd1dQwCza0zHGHeFokmRJOWfz87I/kube-config
```
Copy and paste the first 2 lines into the test terminal.
To run the test successfully, `DOCKER_CERT_PATH` and `KUBECONFIG` must be absolute paths.
 


Do this for all tenants in their respective terminals.



## Step 4: Run Test Script
`cd proxy-test`

In each corresponding window, run the test script, `test_containers.sh`

`test_containers.sh` has 6 possible flags (all optional): <br />
 1. -l (proxy_Location): either local or dev-mon01. Indicates where proxy should be run - on user's local machine, or in the "dev-mon01" data center. For container testing, the proxy runs locally. Local is the default. <br />
 1. -n (network_id): id of the network element to be inspected. Currently not supported; therefore, default is empty. <br />
 1. -t (tenant_id): the tenant id to be used in testing. Default is `test1`. <br />
 1. -k (test_kube): `true` or `false`. Default is true; kube tests will be run. <br />
 1. -c (num_containers): the total number of containers to be created, all without network. If no argument passed, default value is 5. <br />
 1. -p (num_pods): the total number of pods to be created. If no argument is passed, default value is 5. <br />
 1. -a (parallel): `true` or `false`. Default is false; tests will be run in parallel. <br />


Suppose each test wants to examine a lifecycle of 3 containers and 3 pods.
Then, in each corresponding window, run:

```bash
./test_containers.sh -t test1 -c 3 -p 3
./test_containers.sh -t test2 -c 3 -p 3
./test_containers.sh -t test3 -c 3 -p 3
```

## Step 5: Results

Both Docker Swarm & Kubernetes tests will be executed. Summaries of the test results can be found in the `logs` folder, under `<tenant_id>_test_kube_pods_results_<timestamp>.log` for the Kube tests, and `<tenant_id>test_swarm_results_<timestamp>.log` for the Swarm tests.

Each line of the results file will look similar to this:
`20160705.144709,PASS,1,test1,swarm,Test 1,docker ps,0,exit code equals 0,OK`.
Each field is comma-seperated, and represents the following:

1. Timestamp of when the command was executed. Read as: YYYYMMDD.HHMMSS.
2. Result: either `PASS` or `FAIL`.
3. Execution time of the command, in seconds.
4. `<tenant_id>`
5. Test type: either docker `swarm` or `kube`.
6. Test number (sequential).
7. Description of test run.
8. Result code of the command.
9. Expected result.
10. If test passed, will just be `OK`. Otherwise, if fail, further information on what went wrong (the error displayed to the user).
