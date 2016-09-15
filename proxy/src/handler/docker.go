//docker and swarm handler
//
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"auth"       // my auth package
	"conf"       // my conf package
	"httphelper" //my httphelper package

	"github.com/golang/glog"
)

// supported docker api uri prefixes
var dockerPatterns = []string{
	"/containers/",
	"/images/",
	"/exec/",
	"/version",
	"/auth",
	"/_ping",
	//"/build", 	// buildsrvc??
	"/commit", // ?? create img from  container content
	"/info",   // ??  system wide info
	"/events", // ??
	"/networks",
}

// Router based on uri patterns wih simple expressions
var dockerRouter *Router

//called from init() of the handler package, before any requests are handled
func InitDockerHandler() {
	//define routes for api endpoints
	dockerRoutes := []Route{
		NewRoute("DELETE", "/{version}/images/{img}", removeImage),
		NewRoute("DELETE", "/{version}/images/{reg}/{img}", removeImage),
		NewRoute("DELETE", "/{version}/images/{reg}/{ns}/{img}", removeImage),

		NewRoute("GET", "/{version}/images/{img}/json", inspectImage),
		NewRoute("GET", "/{version}/images/{reg}/{img}/json", inspectImage),
		NewRoute("GET", "/{version}/images/{reg}/{ns}/{img}/json", inspectImage),

		NewRoute("GET", "/{version}/images/json", listImages),

		NewRoute("POST", "/{version}/containers/create", createContainer),

		NewRoute("POST", "/{version}/images/create", notSupported),                //pull
		NewRoute("POST", "/{version}/images/{img}/push", notSupported),            //push
		NewRoute("POST", "/{version}/images/{reg}/{img}/push", notSupported),      //push
		NewRoute("POST", "/{version}/images/{reg}/{ns}/{img}/push", notSupported), //push

		// for MVP we are disabling the networ operations
		//	NewRoute("POST", "/{version}/networks/create", createNetwork),
		NewRoute("GET", "/{version}/networks/{name}", inspectNetwork),
		//	NewRoute("DELETE", "/{version}/networks/{name}", removeNetwork),
		//	NewRoute("POST", "/{version}/networks/{name}/connect", connectToNetwork),
		//	NewRoute("POST", "/{version}/networks/{name}/disconnect", disconnectFromNetwork),
		NewRoute("POST", "/{version}/networks/create", notImplemented),
		NewRoute("GET", "/{version}/networks/{name}", notImplemented),
		NewRoute("DELETE", "/{version}/networks/{name}", notImplemented),
		NewRoute("POST", "/{version}/networks/{name}/connect", notImplemented),
		NewRoute("POST", "/{version}/networks/{name}/disconnect", notImplemented),

		NewRoute("*", "*", dockerHandler), //wildcard for forwarding everything else
	}
	dockerRouter = NewRouter(dockerRoutes)
}

// http proxy forwarding with hijack support
// handler for docker/swarm
func DockerEndpointHandler(w http.ResponseWriter, r *http.Request) {
	req_id := conf.GetReqId()
	glog.Infof("------> DockerEndpointHandler triggered, req_id=%s, URI=%s\n", req_id, r.RequestURI)

	// check if uri pattern is accepted
	if !IsSupportedPattern(r.RequestURI, dockerPatterns) {
		glog.Infof("Docker pattern not accepted, req_id=%s, URI=%s", req_id, r.RequestURI)
		NoEndpointHandler(w, r)
		glog.Infof("------ Completed processing of request req_id=%s\n", req_id)
		return
	}

	data, _ := httputil.DumpRequest(r, true)
	glog.Infof("Request dump req_id=%s req_length=%d:\n%s", req_id, len(data), string(data))

	var creds auth.Creds

	// workaround defective sharding in dev-mon
	creds = auth.FileAuth(r)
	if creds.Status == 200 {
		glog.Infof("Authentication from FILE succeeded for req_id=%s status=%d", req_id, creds.Status)
		//glog.Infof("***** creds: %+v", creds)
	} else {
		glog.Errorf("Authentication failed for req_id=%s status=%d", req_id, creds.Status)
		if creds.Status == 401 {
			NotAuthorizedHandler(w, r)
		} else {
			ErrorHandler(w, r, creds.Status)
		}
		glog.Infof("------ Completed processing of request req_id=%s\n", req_id)
		return
	}

	body, _ := ioutil.ReadAll(r.Body)

	//Call conn limiting interceptor(s) pre-processing
	//	if !limit.OpenConn(creds.Container, conf.GetMaxContainerConn()) {
	//		glog.Infof("Max conn limit reached for container...aborting request")
	//		glog.Infof("------ Completed processing of request req_id=%s\n", req_id)
	//		return
	//	}
	//	if !limit.OpenConn(creds.Node, conf.GetMaxNodeConn()) {
	//		glog.Infof("Max conn limit reached for host node...aborting request")
	//		glog.Infof("------ Completed processing of request req_id=%s\n", req_id)
	//		return
	//	}

	//Handle request
	//dockerHandler(w, r, body, creds, nil /*vars*/, req_id)
	dockerRouter.DoRoute(w, r, body, creds, req_id)

	//Call conn limiting interceptor(s) post-processing, to decrement conn count(s)
	//	limit.CloseConn(creds.Container, conf.GetMaxContainerConn())
	//	limit.CloseConn(creds.Node, conf.GetMaxNodeConn())

	glog.Infof("------ Completed processing of request req_id=%s\n", req_id)
}

///////////////////
// route handlers
///////////////////

func notSupported(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	glog.Warningf("Docker pattern not accepted, URI=%s", r.RequestURI)
	NoEndpointHandler(w, r)
}

func notImplemented(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	glog.Warningf("Docker pattern not implemented, URI=%s", r.RequestURI)
	NotImplementedHandler(w, r)
}

func removeImage(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	img := getImageFullnameFromVars(vars)
	if !is_img_valid(img, creds.Reg_namespace) {
		glog.Errorf("Not allowed to access image img=%s namespace=%s req_id=%s", img, creds.Reg_namespace, req_id)
		NotAuthorizedHandler(w, r)
	} else {
		invoke_reg_rmi(w, r, img, creds, req_id)
	}
}

func inspectImage(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	img := getImageFullnameFromVars(vars)
	if !is_img_valid(img, creds.Reg_namespace) {
		glog.Errorf("Not allowed to access image img=%s namespace=%s req_id=%s", img, creds.Reg_namespace, req_id)
		NotAuthorizedHandler(w, r)
	} else {
		invoke_reg_inspect(w, r, img, creds, req_id)
	}
}

func listImages(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	invoke_reg_list(w, r, creds, req_id)
}

func createContainer(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	glog.Infof("createContainer invoked req_id=%s", req_id)
	// extract image
	// disable image inspection
	//	img := get_image_from_container_create(body)
	//	if !is_img_valid(img, creds.Reg_namespace){
	//		Log.Printf("Not allowed to access image img=%s namespace=%s req_id=%s", img, creds.Reg_namespace, req_id)
	//		NotAuthorizedHandler(w, r)
	//		return
	//	}
	// inject X-Registry-Auth header
	//	InjectRegAuthHeader(r, creds)

	net := getNetworkFromContainerCreate(body)
	// if net != "" && net != "default" && net != "bridge"{

	// make the net==none by default:
	if net == "none" || net == "" {
		glog.Info("executing --net=none")
		dockerHandler(w, r, body, creds, vars, req_id)
		return
	}

	if net != "default" {
		//copy body and replace net name by space_id+name
		//body = rewriteNetworkInContainerCreate(body, creds.Space_id)
		ForbiddenOperationHandler(w, r, "Only default network currenlty supported")
	} else {
		//pass through
		body = rewriteNetworkInContainerCreate(body, creds.Space_id)
		dockerHandler(w, r, body, creds, vars, req_id)
	}
}

func createNetwork(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	// extract net name
	name := getNetworkFromNetworkCreate(body)
	if name != "" && name != "default" && name != "bridge" {
		//copy body and replace name by space_id+name
		body = rewriteNetworkInNetworkCreate(body, creds.Space_id)
	}
	dockerHandler(w, r, body, creds, vars, req_id)
}

func inspectNetwork(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	name := vars["{name}"]
	if name != "default" {
		ForbiddenOperationHandler(w, r, "Only default network currently supported")
	} else {
		r.RequestURI = rewriteNetworkUri(r.RequestURI, name, creds.Space_id)
		dockerHandler(w, r, body, creds, vars, req_id)
	}
}

func removeNetwork(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	name := vars["{name}"]
	r.RequestURI = rewriteNetworkUri(r.RequestURI, name, creds.Space_id)
	dockerHandler(w, r, body, creds, vars, req_id)
}

func connectToNetwork(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	name := vars["{name}"]
	r.RequestURI = rewriteNetworkUri(r.RequestURI, name, creds.Space_id)
	dockerHandler(w, r, body, creds, vars, req_id)
}

func disconnectFromNetwork(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {
	name := vars["{name}"]
	r.RequestURI = rewriteNetworkUri(r.RequestURI, name, creds.Space_id)
	dockerHandler(w, r, body, creds, vars, req_id)
}

// default route handler
func dockerHandler(w http.ResponseWriter, r *http.Request, body []byte, creds auth.Creds, vars map[string]string, req_id string) {

	redirect_host := creds.Node
	sp := strings.Split(creds.Node, ":")
	// if port is not provided (Radiant) assign the default swarm port
	if len(sp) < 2 {
		redirect_host = redirect_host + ":" + strconv.Itoa(conf.GetSwarmMasterPort())
		glog.Infof("Assigning proper Swarm  port. Old target: %v, New target: %v", creds.Node, redirect_host)
	}

	redirect_resource_id := creds.Docker_id
	tls_override := creds.Tls_override

	req_UPGRADE := false
	resp_UPGRADE := false
	resp_STREAM := false
	resp_DOCKER := false
	req_LOGS := false

	var err error = nil

	//***** Filter req/headers here before forwarding request to server *****

	if httphelper.IsUpgradeHeader(r.Header) {
		glog.Infof("@ Upgrade request detected\n")
		req_UPGRADE = true
	}
	if is_container_logs_call(r.RequestURI) {
		glog.Infof("@ Logs request detected\n")
		req_LOGS = true
	}

	maxRetries := 1
	backOffTimeout := 0
	if is_container_attach_call(r.RequestURI) {
		//insert delay to allow for completion of container creation on the prior create command
		//time.Sleep(15*time.Second)
		maxRetries = conf.GetMaxRetries()
		backOffTimeout = conf.GetBackOffTimeout()
	}

	glog.Infof("Redirecting to host=%s req_id=%s", redirect_host, req_id)
	var (
		resp *http.Response
		cc   *httputil.ClientConn
	)
	for i := 0; i < maxRetries; i++ {
		resp, err, cc = redirect_random(r, body, redirect_host, redirect_resource_id,
			dockerRewriteUri, tls_override)
		if err == nil {
			break
		}
		glog.Warningf("redirect to host %s failed retry=%d req_id=%s err=%s",
			redirect_host, i, req_id, err)
		if (i + 1) < maxRetries {
			glog.Infof("will sleep before retry secs=%d req_id=%s", backOffTimeout, req_id)
			time.Sleep(time.Duration(backOffTimeout) * time.Second)
		}
	}
	if err != nil {
		glog.Errorf("Error in redirection, will abort req_id=%s err=%v\n", req_id, err)
		msg := "Docker service unavailable or disabled for this shard"
		ErrorHandlerWithMsg(w, r, 503, msg)
		return
	}

	//write out resp
	//now = time.Now()
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
	if httphelper.IsDockerHeader(resp.Header) {
		glog.Infof("@ application/vnd.docker.raw-stream detected\n")
		resp_DOCKER = true
	}

	//***** Filter framework for Interception of commands before forwarding resp to client (1) *****

	proto := strings.ToUpper(httphelper.GetHeader(resp.Header, "Upgrade"))
	if (req_UPGRADE || resp_UPGRADE) && (proto != "TCP") {
		glog.Warningf("Warning: will start hijack proxy loop although Upgrade proto %s is not TCP\n", proto)
	}

	if req_UPGRADE || resp_UPGRADE || resp_STREAM || resp_DOCKER || req_LOGS {
		//resp header is sent first thing on hijacked conn
		w.WriteHeader(resp.StatusCode)

		glog.Infof("starting tcp hijack proxy loop req_id=%s", req_id)
		httphelper.InitProxyHijack(w, cc, req_id, "TCP") // TCP is the only supported proto now
		return
	}
	//If no hijacking, forward full response to client
	w.WriteHeader(resp.StatusCode)

	if resp.Body == nil {
		glog.Infof("\n")
		fmt.Fprintf(w, "\n")
		return
	}

	_DOCKER_CHUNKED_READ_ := false // new feature flag

	if _DOCKER_CHUNKED_READ_ {
		//new code to test
		//defer resp.Body.Close()   // causes this method to not return to caller IF closing while there is still data in Body!
		chunkedRWLoop(resp, w, req_id)

		// TODO extract exec id from resp

	} else {
		resp_body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			glog.Error("Error: error in reading server response body")
			fmt.Fprint(w, "error in reading server response body\n")
			return
		}

		//***** Filter framework for Interception of commands before returning result to client (2) *****
		//Check if Redis caching is required
		//if request uri contains "/container/" and "/exec" then store in Redis the returned exec id (in resp body) and container id (in uri)
		//		if ! creds.Swarm_shard {
		//			//This is needed only in nova-docker case
		//			if is_container_exec_call(r.RequestURI) {
		//				container_id := strip_nova_prefix(redirect_resource_id)
		//				exec_id := get_exec_id_from_response(resp_body)
		//				if exec_id == "" {
		//					glog.Errorf("Error: error in retrieving exec id from response body")
		//				}else {
		//					conf.RedisSetExpire(exec_id, container_id, 60*60)
		//				}
		//			}
		//		}

		//Printout the response body
		bodystr := "Dump Body:\n"
		bodystr += httphelper.PrettyJson(resp_body)
		glog.Info(bodystr)
		/*
			if strings.ToLower(httphelper.GetHeader(resp.Header, "Content-Type")) == "application/json" {
				bodystr += httphelper.PrettyJson(resp_body)
			}else {
				bodystr += string(resp_body)+"\n"
			}
		*/
		//forward server response to calling client
		fmt.Fprintf(w, "%s", resp_body)
	}
	return
}

//Convert /v*/containers/id/*  to  /<docker_api_ver>/containers/<redirect_resource_id>/*
//Convert /v*/exec/id/*  to  /<docker_api_ver>/exec/<redirect_resource_id>/*
func dockerRewriteUri(reqUri string, redirect_resource_id string) (redirectUri string) {
	sl := strings.Split(reqUri, "/")
	if redirect_resource_id == "" {
		//supports /v../containers/json  /v../build  /v../build?foo=bar
		//redirectURI = conf.GetDockerApiVer()+"/"+sl[2]+"/"+sl[3]
		redirectUri = conf.GetDockerApiVer()
		for i := 2; i < len(sl); i++ {
			redirectUri += "/" + sl[i]
		}
	} else {
		//redirectURI = conf.GetDockerApiVer()+"/"+sl[2]+"/"+redirect_resource_id+"/"+sl[4]
		redirectUri = conf.GetDockerApiVer() + "/" + sl[2] + "/" + redirect_resource_id
		for i := 4; i < len(sl); i++ {
			redirectUri += "/" + sl[i]
		}
		//what if there is ?foo=bar in last slice and last slice is resource_id e.g., DELETE /v/containers/123?foo=bar
		if len(sl) <= 4 {
			sl2 := strings.Split(sl[len(sl)-1], "?")
			if len(sl2) > 1 {
				redirectUri += "?" + sl2[1]
			}
		}
	}
	glog.Infof("dockerRewriteURI: '%s' --> '%s'\n", reqUri, redirectUri)
	return redirectUri
}

func strip_nova_prefix(id string) string {
	return strings.TrimPrefix(id, "nova-")
}

func get_exec_id_from_response(body []byte) string {
	type Resp struct {
		Id       string
		Warnings []string
	}
	var resp Resp

	glog.Infof("get_exec_id_from_response: json=%s\n", body)
	err := json.Unmarshal(body, &resp)
	if err != nil {
		glog.Errorf("get_exec_id_from_response: error=%v", err)
		return ""
	}
	glog.Infof("get_exec_id_from_response: Id=%s\n", resp.Id)
	return resp.Id
}

//////////////////////////// Check request URI for a certain call pattern

//return true if it is /<v>/containers/<id>/exec api call
func is_container_exec_call(uri string) bool {
	if strings.Contains(uri, "/containers/") && strings.Contains(uri, "/exec") {
		return true
	} else {
		return false
	}
}

func is_container_attach_call(uri string) bool {
	if strings.Contains(uri, "/containers/") && strings.Contains(uri, "/attach") {
		return true
	} else {
		return false
	}
}

func is_container_logs_call(uri string) bool {
	if strings.Contains(uri, "/containers/") && strings.Contains(uri, "/logs") {
		return true
	} else {
		return false
	}
}

///////////////////////
// network api helpers
///////////////////////

func rewriteNetworkUri(uri, name, space_id string) string {
	fullname := uniqueNetName(name, space_id)
	sl := strings.Split(uri, "/")
	newUri := sl[0]
	for i := 1; i < len(sl); i++ {
		if name == sl[i] {
			newUri += "/" + fullname
		} else {
			newUri += "/" + sl[i]
		}
	}
	return newUri
}

//look for { "name" : "xxx",.... }
func getNetworkFromNetworkCreate(body []byte) (net string) {
	var f interface{}
	err := json.Unmarshal(body, &f)
	if err != nil {
		glog.Errorf("getNetworkFromNetworkCreate: error in json unmarshalling, err=%v", err)
		return
	}
	m := f.(map[string]interface{})
	for k, v := range m {
		if (k == "name") || (k == "Name") {
			net = v.(string)
			glog.Infof("getNetworkFromNetworkCreate: found net=%s", net)
			return
		}
	}
	glog.Warning("getNetworkFromNetworkCreate: did not find name in json body")
	return
}

// look for "{......, "HostConfig":{...., "NetworkMode": "xxxx", ....} }
func getNetworkFromContainerCreate(body []byte) (net string) {
	var f interface{}
	err := json.Unmarshal(body, &f)
	if err != nil {
		glog.Errorf("getNetworkFromContainerCreate: error in json unmarshalling, err=%v", err)
		return
	}
	m := f.(map[string]interface{})
	for k, v := range m {
		if k == "HostConfig" {
			hc := v.(map[string]interface{})
			for kk, vv := range hc {
				if kk == "NetworkMode" {
					net = vv.(string)
					glog.Infof("getNetworkFromContainerCreate: found net=%s", net)
					return
				}
			}
		}
	}
	glog.Warning("getNetworkFromContainerCreate: did not find NetworkMode in json body")
	return
}

func rewriteNetworkInNetworkCreate(body []byte, space_id string) (b []byte) {
	type netCreate struct {
		Name   string
		Driver string
	}
	var nc netCreate
	err := json.Unmarshal(body, &nc)
	if err != nil {
		glog.Errorf("rewriteNetworkInNetworkCreate: Unmarshal error=%v", err)
		b = body
		return
	}
	nc.Name = uniqueNetName(nc.Name, space_id)
	b, err = json.Marshal(&nc)
	if err != nil {
		glog.Errorf("rewriteNetworkInNetworkCreate: Marshal error=%v", err)
		b = body
		return
	}
	glog.Infof("rewriteNetworkInNetworkCreate: unique name=%s", nc.Name)
	return
}

//TODO check if redo using json parsing is more suitable
func rewriteNetworkInContainerCreate(body []byte, space_id string) (b []byte) {
	var sep = []byte("\"NetworkMode\":\"")
	i := bytes.Index(body, sep)
	i += 15                                  //position of net name
	j := bytes.Index(body[i:], []byte("\"")) // j == len of name
	j += i                                   //position of double-quote after net name

	glog.Infof("i=%d j=%d", i, j)
	//var nameBytes []byte
	//nameBytes = make([]byte, j-i+1)
	glog.Infof("**%s**", string(body[i:j]))

	//copy(nameBytes, body[i:j])
	//buf := bytes.NewBuffer(nameBytes)
	//nameString := buf.String()
	nameString := string(body[i:j])
	//nameString = strings.TrimSpace(nameString)
	glog.Infof("nameString=**%s**", nameString)

	fullnameString := uniqueNetName(nameString, space_id)

	newBodyStr := string(body[:i]) + fullnameString + string(body[j:])

	//b = make([]byte, len(newBody))
	b = []byte(newBodyStr)

	glog.Infof("rewriteNetworkInContainerCreate: New Body=**%s**", newBodyStr)

	return
}

func uniqueNetName(net, space string) string {
	return "s" + space + "-" + net
}
