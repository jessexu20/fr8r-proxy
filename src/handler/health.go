package handler

import (
	"fmt"
	"net/http"

	"auth"
	"conf"

	"github.com/golang/glog"
)

var healthRouter *Router

//called from init() of the package
func InitHealthHandler() {
	healthRoutes := []Route{
		NewRoute("GET", "/hjproxy/health", healthHandler),
		NewRoute("GET", "/hjproxy/stats", statsHandler),
		NewRoute("GET", "/hjproxy/_ping/{node}", pingHandler),
		NewRoute("GET", "/hjproxy/_ping/{node}/{port}", pingHandler),
		NewRoute("GET", "/hjproxy/_ping_notls/{node}", pingNotlsHandler),
		NewRoute("GET", "/hjproxy/_ping_notls/{node}/{port}", pingNotlsHandler),
		NewRoute("*", "*", defaultHealthHandler), //wildcard for everything else
	}
	healthRouter = NewRouter(healthRoutes)
}

func HealthEndpointHandler(w http.ResponseWriter, r *http.Request) {
	glog.Infof("HealthEndpointHandler triggered, URI=%s", r.RequestURI)
	healthRouter.DoRoute(w, r, nil /*body*/, auth.Creds{}, "" /*req_id*/)
}

func healthHandler(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	v := conf.GetVerStr()
	fmt.Fprintf(w, "hjproxy up\n%s\n", v)
	glog.Info("hjproxy up ", v)
}

func statsHandler(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	v := conf.GetVerStr()
	n := conf.GetNumServedRequests()
	fmt.Fprintf(w, "hjproxy %s\n", v)
	fmt.Fprintf(w, "This instance served %d non-admin requests\n", n)
	glog.Info("hjproxy ", v)
	glog.Infof("This instance served %d non-admin requests", n)
}

func pingHandler(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	_ping(w, r, false, vars)
}

func pingNotlsHandler(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	_ping(w, r, true, vars)
}

func defaultHealthHandler(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	glog.Infof("Health pattern not accepted, URI=%s", r.RequestURI)
	NoEndpointHandler(w, r)
}

func _ping(w http.ResponseWriter, r *http.Request, tls_override bool, vars map[string]string) {
	//vars contains {node} and optional {port}
	node := vars["{node}"]
	port := conf.GetDockerPort()
	if _, ok := vars["{port}"]; ok {
		port = vars["{port}"]
	}
	redirect_host := node + ":" + port
	resp, err, _ := redirect(r, nil /*body*/, redirect_host, "" /*resource_id*/, pingRewriteUri, tls_override)
	var status int
	if err != nil {
		glog.Errorf("Error in redirection of _ping to %s ... err=%v", redirect_host, err)
		status = 500
	} else {
		status = resp.StatusCode
	}
	if status == 200 {
		glog.Infof("_ping success to host=%s", redirect_host)
		fmt.Fprintf(w, "_ping success to host=%s\n", redirect_host) //returns status 200
	} else {
		//ErrorHandler(w, r, status)
		msg := fmt.Sprintf("_ping failed to host=%s", redirect_host)
		ErrorHandlerWithMsg(w, r, status, msg)
	}
}

func pingRewriteUri(reqUri string, resource_id string) (newReqUri string) {
	newReqUri = conf.GetDockerApiVer() + "/_ping"
	glog.Infof("pingRewriteURI: '%s' --> '%s'", reqUri, newReqUri)
	return newReqUri
}
