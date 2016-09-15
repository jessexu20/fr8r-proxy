package conf

import (
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

//ENV defaults
var env_name = ""
var api_key_header = "X-Tls-Client-Dn"
var use_api_key_header = "false"
var use_api_key_cert = "true"

var docker_port = "8089"
var docker_api_ver = "" // "/v1.18"

var tls_inbound = "false"
var tls_outbound = "false"
var ca_cert_file = "ca.pem"
var server_cert_file = "hjserver.pem"
var server_key_file = "hjserver.key"
var client_cert_file = "hjclient.pem"
var client_key_file = "hjclient.key"

var stub_auth_file = "creds.json"

var registry_location = ""
var registry_admin_password = ""
var consul_ip = ""
var consul_port = 8500

var max_container_conn = 0 //max parallel conn allowed to a container, 0 means no max
var max_node_conn = 0      //max parallel conn allowed to a node, 0 means no max

var max_retries = 10     //max num of times to try connection to server, used in case of attach
var back_off_timeout = 3 //backoff timeout in secs between retries, used in connection with max_retries

var surrogate_ids = "true" // us e surrogate vs random-encoded Ids

// kubernetes parameters:
var kube_port = 443        // Kubernetes master listen port (443 kube-auth)
var kube_authz_port = 8888 // KubeAuthz service port
var kube_admin_key_file = "admin-key.pem"
var kube_admin_cert_file = "admin.pem"
var service_user_template = "system:serviceaccount:$namespace:default"

// swarm parameters:
var swarm_master_port = 2375                      // swarm master port
var swarm_node_port = 2375                        // swarm slave nodes port
var swarm_auth_header = "X-Auth-TenantId"         // header name required by swarm-auth
var swarm_auth_label = "com.ibm.radiant.tenant.0" // the label that is used by swarm to select tenants

// parameter name used for injecting label to kubernetes pods
var annotation_ext_label = "containers-annotations.alpha.kubernetes.io"

var default_listen_port = 6969
var Default_redirect_host = "" //TODO remove this testing default

func init() {
	// configure the logging framework
	flag.Parse()

	log_file_path := os.Getenv("log_file_path")
	if log_file_path != "" {
		flag.Lookup("log_dir").Value.Set(log_file_path)
	}

	log_level := os.Getenv("log_level")
	if log_level != "" {
		flag.Lookup("stderrthreshold").Value.Set(log_level)
	} else {
		flag.Lookup("stderrthreshold").Value.Set("INFO")
	}

	log_verbosity := os.Getenv("log_verbosity")
	if log_verbosity != "" {
		flag.Lookup("v").Value.Set(log_verbosity)
	} else {
		flag.Lookup("v").Value.Set("0")
	}

	// call LoadEnv() here instead of from main package
	LoadEnv()

	// env_name variable must be set for proxy container
	if env_name == "" {
		glog.Fatal("Error: env_name variable must be set for api-proxy container or script")
		os.Exit(1)
	}

	if IsApiKeyHeaderEnabled() == IsApiKeyCertEnabled() {
		glog.Fatal("Error: variable 'use_api_key_header' cannot be equal to 'use_api_key_cert'. Only one must be set to true")
		os.Exit(1)
	}
}

func load_env_var(env_name string, target *string) {
	s := os.Getenv(env_name)
	if s != "" {
		*target = s
	} else {
		s = os.Getenv(strings.ToUpper(env_name))
		if s != "" {
			*target = s
		}
	}
	glog.Infof("load_env_var    : %s=%s", env_name, *target)
}

func load_int_env_var(env_name string, target *int) {
	s := os.Getenv(env_name)
	if s != "" {
		*target, _ = strconv.Atoi(s)
	} else {
		s = os.Getenv(strings.ToUpper(env_name))
		if s != "" {
			*target, _ = strconv.Atoi(s)
		}
	}

	glog.Infof("load_int_env_var: %s=%d", env_name, *target)
}

func LoadEnv() {
	load_env_var("env_name", &env_name)
	load_env_var("api_key_header", &api_key_header)
	load_env_var("use_api_key_header", &use_api_key_header)
	load_env_var("use_api_key_cert", &use_api_key_cert)
	load_env_var("docker_port", &docker_port)
	load_env_var("docker_api_ver", &docker_api_ver)
	load_env_var("tls_inbound", &tls_inbound)
	load_env_var("tls_outbound", &tls_outbound)
	load_env_var("client_cert_file", &client_cert_file)
	load_env_var("client_key_file", &client_key_file)
	load_env_var("ca_cert_file", &ca_cert_file)
	load_env_var("server_cert_file", &server_cert_file)
	load_env_var("server_key_file", &server_key_file)
	load_env_var("stub_auth_file", &stub_auth_file)
	load_env_var("kube_admin_key_file", &kube_admin_key_file)
	load_env_var("kube_admin_cert_file", &kube_admin_cert_file)
	load_env_var("service_user_template", &service_user_template)
	load_env_var("registry_admin_password", &registry_admin_password)
	load_env_var("registry_location", &registry_location)
	load_env_var("surrogate_ids", &surrogate_ids)
	load_int_env_var("max_container_conn", &max_container_conn)
	load_int_env_var("max_node_conn", &max_node_conn)
	load_int_env_var("kube_port", &kube_port)
	load_int_env_var("kube_authz_port", &kube_authz_port)
	load_int_env_var("swarm_master_port", &swarm_master_port)
	load_int_env_var("swarm_node_port", &swarm_node_port)
	load_env_var("swarm_auth_header", &swarm_auth_header)
	load_env_var("swarm_auth_label", &swarm_auth_label)
	load_env_var("annotation_ext_label", &annotation_ext_label)
	load_env_var("consul_ip", &consul_ip)
	load_int_env_var("consul_port", &consul_port)
}

func GetEnvName() string {
	return env_name
}

func GetDefaultListenPort() int {
	return default_listen_port
}

func GetApiKeyHeader() string {
	return api_key_header
}

func GetDockerPort() string {
	return docker_port
}

func GetDockerApiVer() string {
	return docker_api_ver
}

func SetTlsInbound(b bool) {
	if b {
		tls_inbound = "true"
	} else {
		tls_inbound = "false"
	}
}

func IsTlsInbound() bool {
	if tls_inbound == "true" {
		return true
	} else {
		return false
	}
}

func SetTlsOutbound(b bool) {
	if b {
		tls_outbound = "true"
	} else {
		tls_outbound = "false"
	}
}

func IsTlsOutbound() bool {
	if tls_outbound == "true" {
		return true
	} else {
		return false
	}
}

func IsApiKeyHeaderEnabled() bool {
	if use_api_key_header == "true" {
		return true
	}
	return false
}

func IsApiKeyCertEnabled() bool {
	if use_api_key_cert == "true" {
		return true
	}
	return false
}

func GetClientCertFile() string {
	return client_cert_file
}

func GetClientKeyFile() string {
	return client_key_file
}

func GetServerCertFile() string {
	return server_cert_file
}

func GetCaCertFile() string {
	return ca_cert_file
}

func GetServerKeyFile() string {
	return server_key_file
}

func GetStubAuthFile() string {
	return stub_auth_file
}

func GetKadminKeyFile() string {
	return kube_admin_key_file
}

func GetKadminCertFile() string {
	return kube_admin_cert_file
}

func GetServiceUserTemplate() string {
	return service_user_template
}

func GetMaxContainerConn() int {
	return max_container_conn
}

func GetMaxNodeConn() int {
	return max_node_conn
}

func GetMaxRetries() int {
	return max_retries
}

func GetBackOffTimeout() int {
	return back_off_timeout
}

func IsSurrogateIds() bool {
	if surrogate_ids == "true" {
		return true
	} else {
		return false
	}
}

func GetKubePort() int {
	return kube_port
}

func GetKubeAuthzPort() int {
	return kube_authz_port
}

func GetSwarmMasterPort() int {
	return swarm_master_port
}

func GetSwarmNodePort() int {
	return swarm_node_port
}

func GetSwarmAuthHeader() string {
	return swarm_auth_header
}

func GetSwarmAuthLabel() string {
	return swarm_auth_label
}

func GetAnnotationExtLabel() string {
	return annotation_ext_label
}

func GetRegAdminPsswd() string {
	return registry_admin_password
}

func GetRegLocation() string {
	return registry_location
}

func GetConsulIp() string {
	return consul_ip
}

func GetConsulPort() int {
	return consul_port
}
