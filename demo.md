# Open Radiant - IBM Containers

## Demo 1
Demonstrate the separation of Radiant from CCSAPIs authentication. Build TLS
certificates for the specified space id. Add this new space into local
`creds.json` file and reference the newly created TLS certificates based on
dynamically created API key. Use these certificates to send a request to kubernetes  
APIs.

### Get the most recent copy of the Proxy code
 * switch to branch `issue#51` and pull the recent changes
 * compile the code using eclipse with go-plugin
 * start the proxy server `./start_proxy.sh`

### Render new TLS certs:
Let's use the existing `space=017fe867-f756-4b7f-9584-f9cd219a715e`
Execute  `./start_TLS.sh $space` then display the content of the newly updated file:
`cat certs.json`
Copy the `Apikey` for the given space and set as user (below)


### Initialize the kubernetes namespace for this newly created account
```
export user=xxxx
curl -XPOST -H "X-Tls-Client-Dn: /CN=$user" -H "Content-Type: application/json"   localhost:8087/kubeinit
```
Review the console log of the proxy and show the about successfully processing
the local auth file. e.g:
```
2016/06/08 22:38:53.069593 kubeadmin.go:52: This is a AUTH KubeAdmin supported pattern /kubeinit
2016/06/08 22:38:53.069838 kubeadmin.go:58: Authentication from FILE succeeded for req_id=3 status=200
2016/06/08 22:38:53.069856 kubeadmin.go:59: Will not execute CCSAPI auth
```
You can try to execute the same command one more time. It should be still successful,
but the console log will show duplicated effort:
```
2016/06/08 22:38:53.271711 kubeadmin.go:259: Dump Response Body:
{
	"kind": "Status",
	"apiVersion": "v1",
	"metadata": {},
	"status": "Failure",
	"message": "namespaces \"s017fe867-f756-4b7f-9584-f9cd219a715e-default\" already exists",
	"reason": "AlreadyExists",
	"details": {
		"name": "s017fe867-f756-4b7f-9584-f9cd219a715e-default",
		"kind": "namespaces"
	},
	"code": 409
}
2016/06/08 22:38:53.271736 kubeadmin.go:261: <------ req_id=3
2016/06/08 22:38:53.271750 kubeadmin.go:262: Resp StatusCode 409:
2016/06/08 22:38:53.271766 kubeadmin.go:263: Resp Status: 409 Conflict
```
### Execute kubectl transaction
use manual request to demonstrate the TLS certs are used successfully to communicate
with the Kube APIs.


```
curl -XGET -H "X-Tls-Client-Dn: /CN=$user" -H "Content-Type: application/json"   localhost:8087/api
```
This should lead to the following output:
```
{
  "kind": "APIVersions",
  "versions": [
    "v1"
  ],
  "serverAddressByClientCIDRs": [
    {
      "clientCIDR": "0.0.0.0/0",
      "serverAddress": "10.140.171.201:443"
    }
  ]
}
```
### Extra:
### Demonstrate create, list and delete pods using replication controller:

```
# create:
curl -XPOST -H "X-Tls-Client-Dn: /CN=$user" -H "Content-Type: application/json" -H "Accept: application/json, */*" -H "User-Agent: kubectl/v1.2.0 (linux/amd64) kubernetes/d800dca" -d '{"kind":"Deployment","apiVersion":"extensions/v1beta1","metadata":{"name":"test2","creationTimestamp":null,"labels":{"run":"test2"}},"spec":{"replicas":2,"selector":{"matchLabels":{"run":"test2"}},"template":{"metadata":{"creationTimestamp":null,"labels":{"run":"test2"}},"spec":{"containers":[{"name":"test2","image":"mrsabath/web-ms","resources":{"requests":{"memory":"128Mi"}}}]}},"strategy":{}},"status":{}}' localhost:8087/apis/extensions/v1beta1/namespaces/default/deployments

# list:
curl -XGET -H "X-Tls-Client-Dn: /CN=$user" -H "Content-Type: application/json"   localhost:8087/api/v1/namespaces/default/pods

# delete:
curl -XDELETE -H "X-Tls-Client-Dn: /CN=$user" localhost:8087/apis/extensions/v1beta1/namespaces/default/replicasets/test2-xxxx
curl -XDELETE -H "X-Tls-Client-Dn: /CN=$user"  localhost:8087/apis/extensions/v1beta1/namespaces/default/deployments/test2
```

<<<<<<< HEAD
=======
### End of Demo 1
>>>>>>> c36bbe2008e7c29d43ef66fa93b5cd80e049146b
