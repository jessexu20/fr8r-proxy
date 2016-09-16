//kubernetes handler
//
package handler

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"auth"
	"conf"
	"httphelper"

	"github.com/golang/glog"
	"k8s"
)

// supported Kubernetes api uri prefix patterns
// these kube url patterns require namespaces:
var kubePrefixPatterns = []string{
	"/apis/",
	"/api/v1/namespaces/",
	"/api/v1/watch/namespaces/",
	"/api/v1/proxy/namespaces/",
	"/apis/",
	"/api/v1",
	"/apis/extensions",
	"/swaggerapi/",
}

// these kube url patterns don't require namespaces
var kubeExactPatterns = []string{
	"/api",
	"/apis",
	"/version",
}

// Collection of filters to apply to responses from the server to the client
var inboundFilters = k8s.NewFilterCollection()

//called from init() of the package
func InitKubeHandler() {
	// Pod filters
	inboundFilters.AddReplaceFilter("host", "Pod", k8s.NotNilSelector("spec", "nodeName"), "nodeName")
	inboundFilters.AddReplaceFilter("1.1.1.1", "Pod", k8s.NotNilSelector("status", "hostIP"), "hostIP")

	// Event filters
	inboundFilters.AddReplaceFilter("host", "Event", k8s.NotNilSelector("source", "host"), "host")
	inboundFilters.AddRegexFilter("\\sto\\s.+$", " to host", "Event", k8s.IsEqualsSelector("Scheduled", "reason"), "message")
	inboundFilters.AddRegexFilter("\\snode\\s\\([^)]+\\)", " node (host)", "Event", k8s.IsEqualsSelector("FailedScheduling", "reason"), "message")
}

// public handler for Kubernetes
func KubeEndpointHandler(w http.ResponseWriter, r *http.Request) {
	req_id := conf.GetReqId()
	glog.Info("Starting the Kube")
	glog.Infof("------> KubeEndpointHandler triggered, req_id=%s, URI=%s\n", req_id, r.RequestURI)

	// check if URI supported and requires auth.
	if IsExactPattern(r.RequestURI, kubeExactPatterns) {
		glog.Infof("Kube exact pattern accepted, req_id=%s, URI=%s", req_id, r.RequestURI)
	} else if IsSupportedPattern(r.RequestURI, kubePrefixPatterns) {
		glog.Infof("Kube prefix pattern accepted, req_id=%s, URI=%s", req_id, r.RequestURI)
	} else {
		glog.Infof("Kube pattern not accepted, req_id=%s, URI=%s", req_id, r.RequestURI)
		NoEndpointHandler(w, r)
		glog.Infof("------ Completed processing of request req_id=%s\n", req_id)
		return
	}

	glog.Infof("This is a AUTH Kube supported pattern %+v", r.RequestURI)

	// read the credentials from the local file first
	var creds auth.Creds
	creds = auth.FileAuth(r) // So creds should now hold info FOR THAT space_id.
	if creds.Status == 200 {
		glog.Infof("Authentication from FILE succeeded for req_id=%s status=%d", req_id, creds.Status)
	} else {
		glog.Errorf("Authentication from FILE failed for req_id=%s status=%d", req_id, creds.Status)
		if creds.Status == 401 {
			NotAuthorizedHandler(w, r)
		} else {
			ErrorHandler(w, r, creds.Status)
		}
		glog.Infof("------ Completed processing of request req_id=%s\n", req_id)
		return
	}

	// validate the creds
	if creds.Node == "" || creds.Space_id == "" {
		glog.Errorf("Missing data. Host = %v, Space_id = %v", creds.Node, creds.Space_id)
		ErrorHandlerWithMsg(w, r, 404, "Incomplete data received from authentication component")
		return
	}

	// assigning a proper port for Kubernentes
	// the target might or might not contain 'http://', strip it
	redirectTarget := creds.Node
	sp := strings.Split(creds.Node, ":")
	if sp[0] == "http" || sp[0] == "https" {
		redirectTarget = sp[1] + ":" + strconv.Itoa(conf.GetKubePort())
		// strip out the '//' from http://
		redirectTarget = redirectTarget[2:]
	} else {
		redirectTarget = sp[0] + ":" + strconv.Itoa(conf.GetKubePort())
	}

	glog.Infof("Assigning proper Kubernetes port. Old target: %v, New target: %v", creds.Node, redirectTarget)

	// get user certificates from the CCSAPI server
	status, certs := auth.GetCert(r, creds)
	//status, certs := auth.GetCert(r)
	if status != 200 {
		glog.Errorf("Obtaining user certs failed for req_id=%s status=%d", req_id, status)
		ErrorHandler(w, r, creds.Status)
	}
	glog.Infof("Obtaining user certs successful for req_id=%s status=%d", req_id, status)

	// convert the Bluemix space id to namespace
	namespace := auth.GetNamespace(creds.Space_id)
	kubeHandler(w, r, redirectTarget, namespace, req_id, []byte(certs.User_cert), []byte(certs.User_key))
	glog.Infof("------ Completed processing of request req_id=%s\n", req_id)
}

// private handler processing
func kubeHandler(w http.ResponseWriter, r *http.Request, redirect_host string,
	namespace string, req_id string, cert []byte, key []byte) {

	req_UPGRADE := false
	resp_UPGRADE := false
	resp_STREAM := false

	var err error = nil

	data, _ := httputil.DumpRequest(r, true)
	glog.Infof("Request dump of %d bytes:\n%s", len(data), string(data))
	glog.Infof("Redirect host %v\n", redirect_host)

	// sometimes body needs to be modify to add custom labels, annotations
	body, err := kubeUpdateBody(r, namespace)
	if err != nil {
		glog.Errorf("Error %v", err.Error())
		ErrorHandlerWithMsg(w, r, 500, "Error updating Kube body: "+err.Error())
	}

	//***** Filter req/headers here before forwarding request to server *****

	if httphelper.IsUpgradeHeader(r.Header) {
		glog.Infof("@ Upgrade request detected\n")
		req_UPGRADE = true
	}

	maxRetries := 1
	backOffTimeout := 0

	var (
		resp *http.Response
		cc   *httputil.ClientConn
	)
	for i := 0; i < maxRetries; i++ {
		resp, err, cc = redirect_with_cert(r, body, redirect_host, namespace,
			kubeRewriteUri, false, cert, key)

		if err == nil {
			break
		}
		glog.Warningf("redirect retry=%d failed", i)
		if (i + 1) < maxRetries {
			glog.Warningf("will sleep secs=%d before retry", backOffTimeout)
			time.Sleep(time.Duration(backOffTimeout) * time.Second)
		}
	}
	if err != nil {
		glog.Errorf("Error in redirection to server %v, will abort req_id=%s ... err=%v\n", redirect_host, req_id, err)
		msg := "Kubernetes service unavailable or disabled for this shard"
		ErrorHandlerWithMsg(w, r, 503, msg)
		return
	}

	//write out resp
	glog.Infof("<------ req_id=%s\n", req_id)
	//data2, _ := httputil.DumpResponse(resp, true)
	//fmt.Printf("Response dump of %d bytes:\n", len(data2))
	//fmt.Printf("%s\n", string(data2))

	glog.Infof("Resp Status: %s\n", resp.Status)
	glog.Info(httphelper.DumpHeader(resp.Header))

	httphelper.CopyHeader(w.Header(), resp.Header)

	if httphelper.IsUpgradeHeader(resp.Header) {
		glog.Infof("@ Upgrade response detected\n")
		resp_UPGRADE = true
	}
	if httphelper.IsStreamHeader(resp.Header) {
		glog.Infof("@ application/octet-stream detected\n")
		resp_STREAM = true
	}

	//TODO ***** Filter framework for Interception of commands before forwarding resp to client (1) *****

	proto := strings.ToUpper(httphelper.GetHeader(resp.Header, "Upgrade"))
	if (req_UPGRADE || resp_UPGRADE) && (proto != "TCP") {
		glog.Warningf("Warning: will start hijack proxy loop although Upgrade proto %s is not TCP\n", proto)
	}

	if req_UPGRADE || resp_UPGRADE || resp_STREAM {
		//resp header is sent first thing on hijacked conn
		w.WriteHeader(resp.StatusCode)

		glog.Infof("starting tcp hijack proxy loop\n")
		httphelper.InitProxyHijack(w, cc, req_id, "TCP") // TCP is the only supported proto now
		return
	}

	//If no hijacking, forward full response to client

	_KUBE_CHUNKED_READ_ := false // new feature flag

	if _KUBE_CHUNKED_READ_ {
		//new code to test
		//defer resp.Body.Close()   // causes this method to not return to caller IF closing while there is still data in Body!
		w.WriteHeader(resp.StatusCode)
		chunkedRWLoop(resp, w, req_id)
	} else {
		defer resp.Body.Close()
		resp_body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			glog.Errorf("Error: error in reading server response body\n")
			fmt.Fprint(w, "error in reading server response body\n")
			return
		}

		// Get the filtered body
		filtered_body, filtered := inboundFilters.ApplyToJSON(resp_body)

		// Printout the response body
		if filtered {
			glog.Infof("Filtered Body (%d bytes, %d before filtering):\n%s", len(filtered_body), len(resp_body), httphelper.PrettyJson(filtered_body))
		} else {
			glog.Infof("Dump Body (%d bytes):\n%s", len(resp_body), httphelper.PrettyJson(resp_body))
		}

		// Set the new Content-Length header, this has to be done before the first call
		// to Write or WriteHeader (see https://golang.org/pkg/net/http/#ResponseWriter)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(filtered_body)))

		// Forward server response to calling client
		w.WriteHeader(resp.StatusCode)
		bytesWritten, err := w.Write(filtered_body)
		glog.Infof("%d bytes written", bytesWritten)
		if err != nil {
			glog.Errorf("Error: error in writing server response body %v", err)
		}
	}
	return
}

func kubeUpdateBody(r *http.Request, namespace string) (body []byte, err error) {
	body, _ = ioutil.ReadAll(r.Body)
	if r.Method != "POST" {
		// return the original body
		return body, nil
	}

	// convert the body to string
	bodystr := httphelper.PrettyJson(body)
	glog.Infof("Original JSON: %s", bodystr)

	// the request to create pod looks as follow:
	//	 {
	//	  	"kind":"Pod",
	//	  	"apiVersion:"v1",
	//	  	"metadata":{
	//	  		"name":"testtt1",
	//	  		"namespace":"s21f85bc8-5a1a-403a-8a82-cdb757defd72-default",
	//	  		"annotations":{
	//	  			containers-annotations.alpha.kubernetes.io: "{ \"com.ibm.fr8r.tenant.0\": \"stest1-default\",  \"OriginalName\": \"kube-web-server\" }"
	//
	// and the one to create depoloyment (group):
	//  	"kind": "Deployment",
	//		"apiVersion": "extensions/v1beta1",
	//		"metadata": {
	//			"name": "k3",
	//			"creationTimestamp": null,
	//			"labels": {
	//				"run": "k3"
	//			}
	//		},
	//		"spec": {
	//			"replicas": 1,
	//			"selector": {
	//				"matchLabels": {
	//					"run": "k3"
	//				}
	//			},
	//			"template": {
	//				"metadata": {
	//					"creationTimestamp": null,
	//					"annotations":{
	//	  					containers-annotations.alpha.kubernetes.io: "{ \"com.ibm.fr8r.tenant.0\": \"stest1-default\",  \"OriginalName\": \"kube-web-server\" }"
	//					"labels": {
	//						"run": "k3"
	//					}
	//				},}}}

	// get the label names
	auth_label := conf.GetSwarmAuthLabel()
	annot_label := conf.GetAnnotationExtLabel()
	annotation := k8s.KeyValue{Key: annot_label, Value: "{ \"" + auth_label + "\": \"" + namespace + "\" }"}

	kind, err := k8s.KindFromJSON(body)
	if err != nil {
		glog.Warningf("%v", err)
		return body, nil
	}

	var bytes []byte
	if kind.Is("Pod") {
		bytes, err = kind.Inject(annotation, "metadata", "annotations")
	} else if kind.Is("Deployment", "ReplicaSet", "ReplicationController", "Job") {
		bytes, err = kind.Inject(annotation, "spec", "template", "metadata", "annotations")
	} else {
		// We don't need injection
		return body, nil
	}

	if err != nil {
		glog.Errorf("Error injecting the annotation: %v ", err)
		return nil, err
	}

	bodystr = httphelper.PrettyJson(bytes)
	glog.Info("Updated JSON: %s", bodystr)

	return bytes, nil
}

func kubeRewriteUri(reqUri string, namespace string) (redirectUri string) {
	sl := strings.Split(reqUri, "/")
	next := false
	for i := 0; i < len(sl); i++ {
		if next {
			redirectUri += namespace
			next = false
		} else {
			redirectUri += sl[i]
		}
		if sl[i] == "namespaces" {
			next = true
		}
		//if not done
		if i+1 < len(sl) {
			redirectUri += "/"
		}
	}
	glog.Infof("kubeRewriteURI: '%s' --> '%s'\n", reqUri, redirectUri)
	return redirectUri
}
