package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	// _ "net/http/pprof" //for profiling only

	"conf"    // my conf package
	"handler" // my handlers

	"github.com/golang/glog"
)

func init() {
	//logger package TeeLog var initialization and init() will take place before this main package init() is executed
	//conf.LoadEnv() is called in init() of conf package, before this main package init() is executed
	glog.Info(conf.GetVerStr())
}

func main() {
	//init() is auto called first before main(), and after the init() of all imported packages

	listen_port := conf.GetDefaultListenPort()
	//parse args
	nargs := len(os.Args)
	if nargs > 1 {
		listen_port, _ = strconv.Atoi(os.Args[1])
	}
	glog.Infof("Listening on port %d", listen_port)
	if nargs > 2 {
		conf.Default_redirect_host = os.Args[2]
		glog.V(2).Infof("Default upstream server: %s", conf.Default_redirect_host)
	}
	if nargs > 3 {
		if strings.ToLower(os.Args[3]) == "true" {
			conf.SetTlsInbound(true)
			conf.SetTlsOutbound(true)
		}
		if strings.ToLower(os.Args[3]) == "false" {
			conf.SetTlsInbound(false)
			conf.SetTlsOutbound(true)
		}
		glog.V(2).Infof("tls setup: Inbound=%t, Outbound=%t", conf.IsTlsInbound(), conf.IsTlsOutbound())
	}

	//register handlers for supported url paths, can't register same path twice

	// Healthcheck handler
	http.HandleFunc("/hjproxy/", handler.HealthEndpointHandler)

	// set prefix pattern for Kubernetes handler
	http.HandleFunc("/api/", handler.KubeEndpointHandler)
	http.HandleFunc("/api", handler.KubeEndpointHandler)
	http.HandleFunc("/apis", handler.KubeEndpointHandler)
	http.HandleFunc("/apis/", handler.KubeEndpointHandler)
	http.HandleFunc("/version", handler.KubeEndpointHandler)
	http.HandleFunc("/swaggerapi/", handler.KubeEndpointHandler)

	// kubeinit to initialize kube tenant
	http.HandleFunc("/kubeinit", handler.KubeAdminEndpointHandler)

	//Rely on NGINX to route accepted docker/swarm url paths only to hijackproxy
	http.HandleFunc("/", handler.DockerEndpointHandler)
	glog.Infof("All handlers registered")

	/*for profiling only
	go func() {
		Log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	*/

	// init server on any interface + listen_port
	var err error
	if conf.IsTlsInbound() {
		glog.Info("Starting TLS listener service")
		// Here is how started the server earlier, before parsing the user certs:
		//	err = http.ListenAndServeTLS(":"+strconv.Itoa(listen_port), conf.GetServerCertFile(), conf.GetServerKeyFile(), nil)

		// Validate that all the required certs and keys are available (server and kube admin):
		_, err := ioutil.ReadFile(conf.GetServerCertFile())
		if err != nil {
			glog.Error(err)
		}
		_, err = ioutil.ReadFile(conf.GetServerKeyFile())
		if err != nil {
			glog.Error(err)
		}

		_, err = ioutil.ReadFile(conf.GetKadminCertFile())
		if err != nil {
			glog.Error(err)
		}
		_, err = ioutil.ReadFile(conf.GetKadminKeyFile())
		if err != nil {
			glog.Error(err)
		}

		// Here is the new server setup
		caCert, err := ioutil.ReadFile(conf.GetCaCertFile())
		if err != nil {
			glog.Error(err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		// Setup HTTPS client
		tlsConfig := &tls.Config{
			ClientCAs: caCertPool,
			// NoClientCert
			// RequestClientCert
			// RequireAnyClientCert
			// VerifyClientCertIfGiven
			// RequireAndVerifyClientCert
			ClientAuth: tls.RequireAndVerifyClientCert,
		}
		tlsConfig.BuildNameToCertificate()

		server := &http.Server{
			Addr:      ":" + strconv.Itoa(listen_port),
			TLSConfig: tlsConfig,
		}

		server.ListenAndServeTLS(conf.GetServerCertFile(), conf.GetServerKeyFile())
	} else {
		glog.Info("Starting non-TLS listener service")
		err = http.ListenAndServe(":"+strconv.Itoa(listen_port), nil)
	}

	//print something and exit on fatal error
	if err != nil {
		glog.Fatal("Aborting because ListenAndServe could not start: ", err)
	}
}
