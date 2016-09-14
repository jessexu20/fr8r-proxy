package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"conf" // my conf package

	"github.com/golang/glog"
)

// Use a file as authentication credentials store (mainly for trusted test SWARM tenants)
// BlueMix space id is passed in request header as X-Auth-Project-Id header and is used as search key into the file
func FileAuth(r *http.Request) (creds Creds) {

	// let's start with 400 Bad Request
	creds.Status = 400
	Apikey := ""

	if conf.IsApiKeyHeaderEnabled() {
		// configuration specifies that key is passed in the header
		// Apikey = r.Header.Get("X-Tls-Client-Dn")
		Apikey = r.Header.Get(conf.GetApiKeyHeader())
		if Apikey == "" {
			glog.Error("enable_api_key_header=true, but header is missing")
			return
		}
	}

	if conf.IsApiKeyCertEnabled() {
		// configuration specifies that key is passed through the certificate
		if r.TLS == nil || len(r.TLS.PeerCertificates) < 1 {
			glog.Error("Request missing client TLS certificate")
			return
		}
		glog.Infof("Total of TLS certificate(s) found in request: %v", len(r.TLS.PeerCertificates))
		cn := ""

		for _, cert := range r.TLS.PeerCertificates {
			glog.Infof("CN from the client cert: %v", cert.Subject.CommonName)
			cn = cert.Subject.CommonName

			// cert could be CA:
			if len(cn) != 0 && cn != "containers-api-dev.stage1.ng.bluemix.net" {
				break
			}
		}
		if cn == "" {
			return
		}

		Apikey = cn
		creds.Status = 404
		// Apikey := r.Header.Get("X-Tls-Client-Dn")
	}
	fp, err := os.Open(conf.GetStubAuthFile())
	if err != nil {
		glog.Error(err)
		return
	}
	dec := json.NewDecoder(fp)

	for {
		var c Creds
		if err := dec.Decode(&c); err == io.EOF {
			break
		} else if err != nil {
			glog.Error(err)
			break
		}
		// skip if this is not swarm shard nor radiant endpoint
		if c.Endpoint_type != "radiant" && c.Swarm_shard != true {
			continue
		}

		if Apikey == c.Apikey || Apikey == ("/CN="+c.Apikey) {
			space_id := c.Space_id
			creds = c
			//creds.Status = 200  //return the status that is in the auth conf file
			//Set Swarm Authorization header
			namespace := GetNamespace(space_id)
			r.Header.Set(conf.GetSwarmAuthHeader(), namespace)
			return
		}
	}
	glog.Infof("Tenant API key %s not found in %s file", Apikey, conf.GetStubAuthFile())
	//tenant not found in credentials file
	creds.Status = 401
	return
}

func GetNamespace(space_id string) (namespace string) {
	return "s" + space_id + "-default"
}
