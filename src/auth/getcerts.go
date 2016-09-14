package auth

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
)

// '/v3/tlskey' CCSAPI call response
type GetCertResp struct {
	User_cert   string
	User_key    string
	Server_cert string
	Ca_cert     string
}

// forward r header only without body to ccsapi auth endpoint
// return ok=true if resp status is 200, otherwise ok=false

// Editing GetCert to retrieve from local path.
func GetCert(r *http.Request, creds Creds) (status int, certs GetCertResp) {
	glog.Info("Getting certs from local file")
	TLS_path := creds.TLS_path

	certs_bytes, cert_err := ioutil.ReadFile(TLS_path + "/cert.pem")
	if cert_err != nil {
		glog.Errorf("GetCert: Error... %v\n", cert_err)
		return 500, certs
	}
	certs.User_cert = string(certs_bytes)

	glog.Infof("Key path = %s", TLS_path+"/key.pem")
	key_bytes, key_err := ioutil.ReadFile(TLS_path + "/key.pem")
	if key_err != nil {
		glog.Errorf("GetCert: Error... %v\n", key_err)
		return 500, certs
	}
	certs.User_key = string(key_bytes)

	// attempt to create TLS cert based on user's cert and key
	// TODO this is repeated later, so probably need to be removed from here
	_, er := tls.X509KeyPair([]byte(certs.User_cert), []byte(certs.User_key))
	if er != nil {
		glog.Errorf("Error loading client key pair for TLS certificate: %v\n", er)
	} else {
		glog.Info("TLS cert is valid")
	}

	return 200, certs

}

func parse_getCert_Response(body []byte, resp *GetCertResp) error {

	err := json.Unmarshal(body, resp)
	if err != nil {
		glog.Errorf("parse_getCert_Response: error=%v", err)
		return err
	}
	s := fmt.Sprintf("parse_getCert_Response: cert=%s key=%s ", resp.User_cert, resp.User_key)
	glog.Infof("Retrieved certs from CCSAPI: %v", s)

	// attempt to create TLS cert based on user's cert and key
	// TODO this is repeated later, so probably need to be removed from here
	_, er := tls.X509KeyPair([]byte(resp.User_cert), []byte(resp.User_key))
	if er != nil {
		glog.Errorf("Error loading client key pair for TLS certificate: %v\n", er)
	} else {
		glog.Info("TLS cert is valid")
	}
	return nil
}
