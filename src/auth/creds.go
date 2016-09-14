package auth

//Creds struct returned to calling handler after authenticating with CCSAPI
type Creds struct {
	Status        int //200 is auth success
	Node          string
	Docker_id     string //docker resource id (e.g., exec id)
	Container     string
	Tls_override  bool   //Don't use tls to connect to this node
	Space_id      string //Bluemix space
	Reg_namespace string //Reg namespace
	Apikey        string
	Orguuid       string
	Userid        string
	Swarm_shard   bool
	Endpoint_type string // radiant, kraken, swarm etc.
	TLS_path      string
}
