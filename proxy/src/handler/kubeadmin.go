//kubernetes handler
//
package handler

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"

	"auth"
	"conf"
	"httphelper"

	"github.com/golang/glog"
)

// supported Kubernetes api uri prefix patterns
// these kube url patterns require namespaces:
var kubeAdminPrefixPatterns = []string{
	"/kubeinit/",
}

// these kube url patterns don't require namespaces
var kubeAdminExactPatterns = []string{
	"/kubeinit",
}

//called from init() of the package
func InitKubeAdminHandler() {

}

// public handler for Kubernetes
func KubeAdminEndpointHandler(w http.ResponseWriter, r *http.Request) {
	req_id := conf.GetReqId()
	glog.Infof("------> KubeAdminEndpointHandler triggered, req_id=%s, URI=%s\n", req_id, r.RequestURI)

	// check if URI supported and requires auth.
	if IsExactPattern(r.RequestURI, kubeAdminExactPatterns) {
		glog.Infof("KubeAdmin exact pattern accepted, req_id=%s, URI=%s", req_id, r.RequestURI)
	} else if IsSupportedPattern(r.RequestURI, kubeAdminPrefixPatterns) {
		glog.Infof("KubeAdmin prefix pattern accepted, req_id=%s, URI=%s", req_id, r.RequestURI)
	} else {
		glog.Infof("KubeAdmin pattern not accepted, req_id=%s, URI=%s", req_id, r.RequestURI)
		NoEndpointHandler(w, r)
		glog.Infof("------ Completed processing of request req_id=%s\n", req_id)
		return
	}

	glog.Infof("This is a AUTH KubeAdmin supported pattern %+v", r.RequestURI)

	// read the credentials from the local file first
	var creds auth.Creds
	creds = auth.FileAuth(r)
	if creds.Status == 200 {
		glog.Infof("Authentication from FILE succeeded for req_id=%s status=%d", req_id, creds.Status)
	} else if creds.Status == 401 {
		NotAuthorizedHandler(w, r)
		return
	} else {
		ErrorHandler(w, r, creds.Status)
		return
	}
	glog.Infof("------ Completed processing of request req_id=%s\n", req_id)

	// validate the creds
	if creds.Node == "" || creds.Space_id == "" {
		glog.Errorf("Missing data. Host = %v, Namespace = %v", creds.Node, creds.Space_id)
		ErrorHandlerWithMsg(w, r, 404, "Incomplete data received from CCSAPI server")
		return
	}

	// assigning a proper port for Kubernentes
	// the target might or might not contain 'http://', strip it
	redirectTarget := creds.Node
	// first remove the http://, https://
	sp := strings.Split(creds.Node, ":")
	if sp[0] == "http" || sp[0] == "https" {
		redirectTarget = sp[1] + ":" + strconv.Itoa(conf.GetKubePort())
		// strip out the '//' from http://
		redirectTarget = redirectTarget[2:len(redirectTarget)]
	}
	// then remove the port
	sp = strings.Split(redirectTarget, ":")
	kubeTarget := sp[0] + ":" + strconv.Itoa(conf.GetKubePort())
	kubeAuthzTarget := sp[0] + ":" + strconv.Itoa(conf.GetKubeAuthzPort())

	glog.Infof("Assigning proper Kubernetes port. Old target: %v, New target: %v", creds.Node, redirectTarget)

	er, ac := getAdminConn(kubeTarget)
	if er != nil {
		glog.Errorf("Error in KubeAdminEndpointHandler, will abort req_id=%s ... err=%v\n", req_id, er)
		ErrorHandlerWithMsg(w, r, 500, "Error getting the Admin Connection.")
		return
	}

	kubeAdminHandler(w, r, kubeTarget, kubeAuthzTarget, creds, req_id, ac)
	glog.Infof("------ Completed processing of request req_id=%s\n", req_id)
}

// private handler processing
func kubeAdminHandler(w http.ResponseWriter, r *http.Request, kubeTarget string,
	kubeAuthzTarget string, creds auth.Creds, req_id string, ac *httputil.ClientConn) {

	glog.Infof("Processing space_id: %v", creds.Space_id)
	namespace := auth.GetNamespace(creds.Space_id)
	body, _ := ioutil.ReadAll(r.Body)
	glog.Infof("Request Body %s", body)

	// execute new request to create namespace
	resp, err := kubeCreateNamespaceRequest(r, kubeTarget, namespace, req_id, ac)
	if err != nil {
		glog.Errorf("Error executing kubeCreateNamespaceRequest: %v", err)
	}
	glog.Infof("Response from kube: %+v", resp)

	switch resp.StatusCode {
	case 201:
		glog.Infof("Namespace %s successuflly created", namespace)
	case 409:
		glog.Infof("Namespace %s already exists ", namespace)
	default:
		glog.Infof("Case default")
		ErrorHandlerWithMsg(w, r, 500, "Error creating K8s namespace: "+namespace)
		return
	}
	resp, err = kubeSetQuotaRequest(r, kubeTarget, namespace, req_id, ac)
	if err != nil {
		glog.Infof("Error executing kubeSetQuotaRequest: %v", err)
	}
	switch resp.StatusCode {
	case 201:
		glog.Infof("Quota successfully set for amespace %s", namespace)
	case 403:
		glog.Infof("Quota already set for namespace %s", namespace)
	default:
		glog.Infof("Case default")
		ErrorHandlerWithMsg(w, r, 500, "Error setting quota for namespace: "+namespace)
		return
	}
	resp, err = kubeSetLimitsRequest(r, kubeTarget, namespace, req_id, ac)
	if err != nil {
		glog.Infof("Error executing kubeSetLimitsRequest: %v", err)
	}
	switch resp.StatusCode {
	case 201:
		glog.Infof("Limits successfully set for amespace %s", namespace)
	case 409:
		glog.Infof("Limits already set for namespace %s", namespace)
	default:
		glog.Infof("Case default")
		ErrorHandlerWithMsg(w, r, 500, "Error setting limits for namespace: "+namespace)
		return
	}

	// TODO this should called a separate handler 'useradmin'
	resp, err = kubeCreateUserRequest(r, kubeAuthzTarget, namespace, creds.Apikey, req_id, ac)
	if err != nil {
		glog.Infof("Error executing kubeCreateUserRequest: %v", err)
	}
	//w http.ResponseWriter, r *http.Request, status int
	switch resp.StatusCode {
	case 200:
		glog.Infof("User successfully created for namespace %s", namespace)
	case 201:
		glog.Infof("User successfully created for namespace %s", namespace)
	case 409:
		glog.Infof("User already set for namespace %s", namespace)
	default:
		glog.Infof("Case default")
		ErrorHandlerWithMsg(w, r, 500, "Error creating a user: "+creds.Apikey+" for namespace: "+namespace)
		return
	}

	// using service user template, add services user to the namespace
	//service_user_template := "system:serviceaccount:$namespace:default"
	service_user_template := conf.GetServiceUserTemplate()
	service_user := strings.Replace(service_user_template, "$namespace", namespace, 1)
	resp, err = kubeCreateUserRequest(r, kubeAuthzTarget, namespace, service_user, req_id, ac)
	if err != nil {
		glog.Infof("Error executing kubeCreateUserRequest: %v", err)
	}
	//w http.ResponseWriter, r *http.Request, status int
	switch resp.StatusCode {
	case 200:
		glog.Infof("Service user successfully created for namespace %s", namespace)
		OkHandler(w, r, 200)
	case 201:
		glog.Infof("Service user successfully created for namespace %s", namespace)
		OkHandler(w, r, 200)
	case 409:
		glog.Infof("Service user already set for namespace %s", namespace)
		OkHandler(w, r, 200)
	default:
		glog.Infof("Case default")
		ErrorHandlerWithMsg(w, r, 500, "Error creating a service user: "+service_user+" for namespace: "+namespace)
		return
	}
}

func kubeCreateNamespaceRequest(r *http.Request, target_host string, namespace string, req_id string, ac *httputil.ClientConn) (resp *http.Response, err error) {
	new_body := "{\"kind\":\"Namespace\",\"apiVersion\":\"v1\",\"metadata\":{\"name\":\"" + namespace
	new_body += "\",\"creationTimestamp\":null},\"spec\":{},\"status\":{}}"
	glog.Infof("Request Body: %s", new_body)
	request := "/api/v1/namespaces"
	return kubeGenericRequest("POST", r, target_host, request, namespace, []byte(new_body), req_id, ac)
}

func kubeSetQuotaRequest(r *http.Request, target_host string, namespace string, req_id string, ac *httputil.ClientConn) (resp *http.Response, err error) {
	new_body := "{\"kind\":\"ResourceQuota\",\"apiVersion\":\"v1\",\"metadata\":{\"namespace\":\"" + namespace
	new_body += "\",\"name\":\"quota\",\"creationTimestamp\":null},\"spec\":{\"hard\":{\"resourcequotas\":\"1\",\"secrets\":\"10\","
	new_body += "\"services\":\"10\"}},\"status\":{}}"
	glog.Infof("Request Body: %s", new_body)
	request := "/api/v1/namespaces/" + namespace + "/resourcequotas"
	return kubeGenericRequest("POST", r, target_host, request, namespace, []byte(new_body), req_id, ac)
}

func kubeSetLimitsRequest(r *http.Request, target_host string, namespace string, req_id string, ac *httputil.ClientConn) (resp *http.Response, err error) {
	new_body := "{\"kind\":\"LimitRange\",\"apiVersion\":\"v1\",\"metadata\":{\"namespace\":\"" + namespace
	new_body += "\",\"name\":\"limits\",\"creationTimestamp\":null},\"spec\":{\"limits\":[{\"type\":\"Pod\","
	new_body += "\"max\":{\"cpu\":\"8\",\"memory\":\"112Gi\"},\"min\":{\"cpu\":\"200m\",\"memory\":\"64Mi\"}},"
	new_body += "{\"type\":\"Container\",\"max\":{\"cpu\":\"8\",\"memory\":\"112Gi\"},\"min\":{\"cpu\":\"200m\","
	new_body += "\"memory\":\"64Mi\"},\"default\":{\"cpu\":\"400m\",\"memory\":\"128Mi\"},\"defaultRequest\":"
	new_body += "{\"cpu\":\"200m\",\"memory\":\"64Mi\"},\"maxLimitRequestRatio\":{\"cpu\":\"2\",\"memory\":\"2\"}}]}}"

	glog.Infof("Request Body: %s", new_body)
	request := "/api/v1/namespaces/" + namespace + "/limitranges"
	return kubeGenericRequest("POST", r, target_host, request, namespace, []byte(new_body), req_id, ac)
}

// TODO this method should be moved to a separate handler 'useradmin'
func kubeCreateUserRequest(r *http.Request, target_host string, namespace string, user string,
	req_id string, ac *httputil.ClientConn) (resp *http.Response, err error) {
	// user administration is done by a different APIs, so create a new ClientConn and new request
	er, uac := getAdminConn(target_host)
	if er != nil {
		glog.Errorf("Error geting AdminCreds: %s", er)
		return nil, er
	}
	// create new user and request access to the namespace
	request := "/user/" + user + "/" + namespace
	return kubeGenericRequest("PUT", r, target_host, request, namespace, nil, req_id, uac)
}

// private handler processing
func kubeGenericRequest(method string, r *http.Request, target_host string,
	request string, namespace string, body []byte, req_id string, ac *httputil.ClientConn) (resp *http.Response, err error) {

	req, _ := http.NewRequest(method, "https://"+target_host+request, bytes.NewReader(body))
	req.Header = r.Header
	req.URL.Host = target_host

	glog.Infof("Executing request to server=%s URI=%s  ...", target_host, req.RequestURI)
	glog.Infof("Request: %+v", req)
	resp, err = ac.Do(req)
	if err != nil {
		glog.Errorf("Error on execution %v", err)
	}
	glog.Infof("Response from kubeadmin: %+v", resp)
	resp_body, err := ioutil.ReadAll(resp.Body)
	//Printout the response body
	bodystr := "Dump Response Body:\n"
	bodystr += httphelper.PrettyJson(resp_body)
	glog.Info(bodystr)
	//write out resp
	glog.Infof("<------ req_id=%s\n", req_id)
	glog.Infof("Resp StatusCode %d: ", resp.StatusCode)
	glog.Infof("Resp Status: %s\n", resp.Status)
	glog.Info(httphelper.DumpHeader(resp.Header))

	return resp, err
}

// TODO this method is not really used. Consider updating the code to use it or remove
func kubeAdminRewriteUri(reqUri string, namespace string) (redirectUri string) {
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
