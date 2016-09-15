# Fr8r API Wrapper - Proxy
Proxy is the component of Fr8r that intercepts the communication between
the clients (Docker or Kubernetes) and the Fr8r cluster, using HTTP session
hijacking. It validates the tenant and provided TLS certificates. The complete
list of features:
*Completed items are marked*
- [x] [Single integration point](#single-integration-point-for-all-the-services) for all the services
- [x] Handles the multi-tenant authentication framework to support various auth interfaces
- [x] [Supports common certification](#central-multi-tenant-authentication-support), creates tenants and their TLS certificates
- [x] Stateless microservice deployed as a container
- [x] Handles re-direct to appropriate framework API handler
- [] [Validates and filters the requests](#request-validation-and-filtering-response-masquerading); Masquerades the internal information that should not be public
- [] Supports [sharding](#sharding-support)
- [] Can execute quota validation across multiple frameworks and shards
- [x] Executes annotation injections to support viewing K8s pods via Docker interface
- [] API version mapping
- [] API Extensions:
  - [] [Quota management](#shared-quota-and-limits-management)
  - [] Health checking, status, HA and informational APIs
  - [] Metering and chargeback management
- [] Prototyping new features (e.g. Powerstrip for Docker)

## Overview of Fr8r Proxy
![Image of Proxy](media/2016-07.Fr8rProxy.png)
Fr8r administrator creates TLS certificate and key for each new Fr8r
user and makes them available to the user. User configures the `kubectl` and `docker`
clients with the provided TLS certificates and the IP address of the proxy.
Each user request received by proxy is redirected to specific handler based on
request URL. Provided TLS certificates are validated and matched with the authentication
module, FileAuth by default. If authentication is successful, the request is
redirected to appropriate shard and port. There could be one or many shards
managed by a single proxy. 



## Fr8r Proxy Details
![Image of Proxy details](media/2016-05.Proxy-details.png)

## Single integration point for all the services
* Proxy is a single point that integrates APIs for all supported frameworks
* Inspects the request format and redirects to appropriate service
* Knows the VIP and port

## Central Multi-tenant Authentication Support
* Single authentication mechanism for all the supported frameworks
* TLS Certification management
* Scripts to create new API keys and TLS certs
* Revoking Certification
* Includes:
  * Tenant
  * User
  * API key
* Pluggable framework:
  * File Auth
  * LDAP (to be created)

## Request Validation and Filtering; Response Masquerading
* Filter the requests or attributes
* To support multi-tenancy, translates the attributes: e.g. `default` network  `s{tenant_id}_default`
* Validates if all the required fields are set
* Prevents user from executing code that is not authorized or not ready
* Possibly removes or masquarades the detailed information that might not be exposed to the end user e.g. internal hostname or IP of the HA cluster member

## Sharding Support
* Shard –instance of Fr8r cluster
* Shard might function as Availability Zone
* Tenant is assigned to a specific shard using VIP
* Allows switching tenants between shards
* Frameworks can be divided to separate shards:
  * Vanilla Kubernetes only shard
  * Kubernetes and Swarm on Mesos
  * Swarm or Swarm-Auth

## Shared Quota and Limits Management
* Single quota across multiple frameworks, shards, clusters
* Keep track of used quota and limits
  * Separate microservice
  * Extension library (with common repository)
* API Extensions for quota and limits management
