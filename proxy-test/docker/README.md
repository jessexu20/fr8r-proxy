# Testing, for API Proxy and tests running as a Container
Steps below explain configuration and setup to run Swarm & Kubernetes testing as a container, while API Proxy is running as a container.

Note: Steps below detail multi-user testing. Testing can also be done with a single-user.

## Step 1: Build the Docker image

Move to the `proxy-test` directory and run the build script
```bash
cd ~/workspace/fr8r/proxy/proxy-test
# dev-vbox is the env name for the tiny example
./docker/build.sh -e dev-vbox
```

The build script has 3 possible flags (`-e` is the only one required):
 1. `-e <environment>`: the environment name.
 2. `-d <docker_version>`: the Docker version that will be installed.
 3. `-k <kubectl_version>`: the Kubernetes version that will be installed.

## Step 2: Certificates

After starting the API Proxy, execute the script to create test tenants:
```bash
docker exec api-proxy /api-proxy/create_tenant.sh test1 radiant01 192.168.10.2
docker exec api-proxy /api-proxy/create_tenant.sh test2 radiant01 192.168.10.2
docker exec api-proxy /api-proxy/create_tenant.sh test3 radiant01 192.168.10.2
```
To view all the accounts valid for this proxy: 
```bash
docker exec api-proxy cat /opt/tls_certs/creds.json
```
The certificate creation script will output few export statements. For example:
```bash
Generating API key of length 48
Generated API key: xoaI1UGKUA4neu6Tubv67nh7XSBmubuVYPrC3MA3E4WXETOX
Creating certificates
Generating RSA private key, 2048 bit long modulus
.+++
.........................................+++
e is 65537 (0x10001)
Signature ok
subject=/CN=xoaI1UGKUA4neu6Tubv67nh7XSBmubuVYPrC3MA3E4WXETOX
Getting CA Private Key
Writing to creds.json
Certificates created for Apikey xoaI1UGKUA4neu6Tubv67nh7XSBmubuVYPrC3MA3E4WXETOX
Located at /opt/tls_certs/radiant01/xoaI1UGKUA4neu6Tubv67nh7XSBmubuVYPrC3MA3E4WXETOX

NOTE: the following commands must be executed from fr8r/proxy directory

# Setup docker environment:
export DOCKER_HOST=localhost:6969
export DOCKER_TLS_VERIFY=1
export DOCKER_CERT_PATH=~/.fr8r/envs/dev-vbox/radiant01/xoaI1UGKUA4neu6Tubv67nh7XSBmubuVYPrC3MA3E4WXETOX

# Setup kubernetes environment:
export KUBECONFIG=~/.fr8r/envs/dev-vbox/radiant01/xoaI1UGKUA4neu6Tubv67nh7XSBmubuVYPrC3MA3E4WXETOX/kube-config
```
For each user you need to remember the API key (e.g. `xoaI1UGKUA4neu6Tubv67nh7XSBmubuVYPrC3MA3E4WXETOX`).

## Step 3: Run the tests

The test image has 6 possible flags (all optional except for `-t`):
 1. `-l <proxy_location>`: either local or dev-mon01. Indicates where proxy should be run - on user's local machine, or in the "dev-mon01" data center. For container testing, the proxy runs locally. Local is the default.
 2. `-n <network_id>`: id of the network element to be inspected. Currently not supported; therefore, default is empty.
 3. `-t <shard>:<tenant_id>:<api_key>`: you can pass as many tenants as you want.
 4. `-k <test_kube?>`: `true` or `false`. Default is true; kube tests will be run. 
 5. `-c <num_containers>`: the total number of containers to be created, all without network. If no argument passed, default value is 5.
 6. `-p <num_pods>`: the total number of pods to be created. If no argument is passed, default value is 5.
 7. `-a <parallel?>`: `true` or `false`. Default is false; tests will be run in parallel.

Suppose each test wants to examine a lifecycle of 3 containers and 3 pods.
Then run:
```bash
./docker/run.sh -c 3 -p 3 \
                -t radiant01:test1:nVdIhfJKFtEM2G9hpkJI5EVZ5VGeFNLBoFBA2B6zJqaSZ71W \
                -t radiant01:test2:JpjnQqnDQQCnNlc9bWJJoznhjr4awLdVe10B45LRCE31CqDh \
                -t radiant01:test3:PBJ7ZKbi0hDSBFStFY5K9TnNdojEhUZ1goE1Swn3G6fle5iR
```

#### Running on Continuous Integration (CI)
When running on an environment with `CI=true` (e.g. Travis) `./docker/run.sh` can be invoked without any tenant, in this case it will run the tests for all the tenants. To override this behaviour al least one tenant must be provided.

## Step 4: Results

Both Docker Swarm & Kubernetes tests will be executed. Summaries of the test results will be printed to console.
```
The env is dev-vbox
The log tag is 2016-08-16_02-11-29
Completed: 3/3

Test results for tenant test1:
Docker Containers Test summary:
Total = 46 tests
Passed = 46 tests
Failed = 0 tests
Total time = 55 sec
Kube Test summary:
Total = 22 tests
Passed = 22 tests
Failed = 0 tests
Total time = 21 sec

Test results for tenant test2:
Docker Containers Test summary:
Total = 46 tests
Passed = 46 tests
Failed = 0 tests
Total time = 54 sec
Kube Test summary:
Total = 22 tests
Passed = 22 tests
Failed = 0 tests
Total time = 21 sec

Test results for tenant test3:
Docker Containers Test summary:
Total = 46 tests
Passed = 46 tests
Failed = 0 tests
Total time = 54 sec
Kube Test summary:
Total = 22 tests
Passed = 22 tests
Failed = 0 tests
Total time = 21 sec

0 failed tests.
```

To show the full log report run:
```bash
./docker/log.sh --latest
```

Otherwise it's possible to print a specific log identified by the log tag (the log timestamp):
```bash
./docker/log.sh -t 2016-08-16_02-11-29
```

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
