package handler

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"

	"conf"

	"github.com/golang/glog"
)

func init() {
	//call initializers for all handlers
	InitDockerHandler()
	InitKubeHandler()
	InitHealthHandler()
}

//called by golang before init() of main packagev
func init() {

}

//Forward req to server
//when tls_override=true tls is disabled for the current request being processed only
//The override is a directive received from ccsapi getHost, for a swarm request when swarm master does not support tls
func redirect(r *http.Request, body []byte, redirect_host string, redirect_resource_id string,
	rewriteURI func(uri string, resource string) string, tls_override bool) (*http.Response, error, *httputil.ClientConn) {

	var cc *httputil.ClientConn

	c, err := net.Dial("tcp", redirect_host)
	if err != nil {
		// handle error
		glog.Errorf("Error connecting to server=%s, %v", redirect_host, err)
		return nil, err, nil
	}

	if conf.IsTlsOutbound() && !tls_override {
		glog.Infof("Excuting TLS redirect")
		cert, er := tls.LoadX509KeyPair(conf.GetClientCertFile(), conf.GetClientKeyFile())
		if er != nil {
			glog.Errorf("Error loading client key pair, %v", er)
			return nil, err, nil
		}
		c_tls := tls.Client(c, &tls.Config{InsecureSkipVerify: true, Certificates: []tls.Certificate{cert}})
		cc = httputil.NewClientConn(c_tls, nil)
	} else {
		cc = httputil.NewClientConn(c, nil)
	}

	req, _ := http.NewRequest(r.Method, "http://"+redirect_host+rewriteURI(r.RequestURI, redirect_resource_id),
		bytes.NewReader(body))
	req.Header = r.Header
	//req.Host = redirect_host
	req.URL.Host = redirect_host

	glog.Infof("will forward request to server=%s ...", redirect_host)
	resp, err := cc.Do(req)
	glog.Infof("Response from redirect_host: %v", resp)
	return resp, err, cc
}

//Forward req to server
//when tls_override=true tls is disabled for the current request being processed only
//The override is a directive received from ccsapi getHost, for a swarm request when swarm master does not support tls
func redirect_with_cert(r *http.Request, body []byte, redirect_host string, redirect_resource_id string,
	rewriteURI func(uri string, resource string) string, tls_override bool, cert []byte,
	key []byte) (*http.Response, error, *httputil.ClientConn) {

	var cc *httputil.ClientConn

	c, err := net.Dial("tcp", redirect_host)
	if err != nil {
		// handle error
		glog.Errorf("Error connecting to server=%s, %v", redirect_host, err)
		return nil, err, nil
	}

	if conf.IsTlsOutbound() && !tls_override {
		var tlscert tls.Certificate
		var er error
		glog.Infof("Excuting TLS redirect")
		tlscert, er = tls.X509KeyPair([]byte(cert), []byte(key))
		if er != nil {
			glog.Errorf("Error loading client key pair, %v", er)
			return nil, err, nil
		}
		c_tls := tls.Client(c, &tls.Config{InsecureSkipVerify: true, Certificates: []tls.Certificate{tlscert}})
		cc = httputil.NewClientConn(c_tls, nil)
	} else {
		cc = httputil.NewClientConn(c, nil)
	}

	req, _ := http.NewRequest(r.Method, "https://"+redirect_host+rewriteURI(r.RequestURI, redirect_resource_id),
		bytes.NewReader(body))
	req.Header = r.Header
	req.URL.Host = redirect_host
	glog.Infof("will forward request to server=%s ...", redirect_host)
	glog.Infof("request URL: %+v", req.URL)
	glog.Infof("all request: %+v", req)
	resp, err := cc.Do(req)
	return resp, err, cc
}

// Assumes redirect_host is a list of comma separated host:port pairs
// selects a target node randomly and calls redirect
// if call fails try other targets until success or exhausting targets
func redirect_random(r *http.Request, body []byte, redirect_host string, redirect_resource_id string,
	rewriteURI func(uri string, resource string) string, tls_override bool) (resp *http.Response, err error, cc *httputil.ClientConn) {

	// get list of host:port pairs
	nodes := strings.Split(redirect_host, ",")
	num_nodes := len(nodes)
	glog.Infof("redirect_random num_nodes=%d nodes=%s", num_nodes, nodes)

	// pick random target
	var target int
	t, e := rand.Int(rand.Reader, big.NewInt(int64(num_nodes)))
	if e != nil {
		glog.Error("error in rand num generator:", e)
		target = 0
		num_nodes = 1
	} else {
		target = int(t.Int64())
	}

	// call redirect
	redirect_host = nodes[target]
	glog.Infof("redirect_random: node=%s target=%d", redirect_host, target)
	resp, err, cc = redirect(r, body, redirect_host, redirect_resource_id, rewriteURI, tls_override)
	if err == nil {
		return
	}

	// on failure loop through rest of targets until success
	for i := 1; i < num_nodes; i++ {
		target += i
		target = target % num_nodes
		redirect_host = nodes[target]
		glog.Infof("redirect_random: node=%s target=%d", redirect_host, target)
		resp, err, cc = redirect(r, body, redirect_host, redirect_resource_id, rewriteURI, tls_override)
		if err == nil {
			break
		}
		glog.Infof("redirect_random: redirect failed  node=%s err=%s", redirect_host, err)
	}

	return
}

func getAdminConn(redirect_host string) (error, *httputil.ClientConn) {
	glog.Infof("Loading Kube Admin certifcates")
	var cc *httputil.ClientConn

	c, err := net.Dial("tcp", redirect_host)
	if err != nil {
		// handle error
		glog.Errorf("Error connecting to server=%s, %v", redirect_host, err)
		return err, nil
	}
	admincert, err := getAdminCert()
	if err != nil {
		glog.Errorf("Error getting admin certs for host=%s: %v", redirect_host, err)
		return err, nil
	}
	c_tls := tls.Client(c, &tls.Config{InsecureSkipVerify: true, Certificates: []tls.Certificate{admincert}})
	cc = httputil.NewClientConn(c_tls, nil)
	return nil, cc
}

func getAdminCert() (tls.Certificate, error) {
	var admincert tls.Certificate
	var er error
	admincert, er = tls.LoadX509KeyPair(conf.GetKadminCertFile(), conf.GetKadminKeyFile())
	if er != nil {
		glog.Errorf("Error loading kube admin key pair, %v", er)
		return admincert, er
	}
	return admincert, nil
}
