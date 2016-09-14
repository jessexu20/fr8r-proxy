package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"auth"
	"conf"
	"httphelper"

	"github.com/golang/glog"
)

var _REG_ADMIN_CREDS_ bool = true //feature flag

func InjectRegAuthHeader(r *http.Request, creds auth.Creds) {
	//Use admin base64 encoded psswd
	//tok := conf.GetRegAuthToken()

	var user, psswd string
	if _REG_ADMIN_CREDS_ {
		user = "admin"
		psswd = conf.GetRegAdminPsswd()
	} else {
		user = "apikey"
		psswd = creds.Apikey
	}
	reg := conf.GetRegLocation()

	//create X-Registry-Auth object out of apikey in creds
	// {"username":"admin","password":"230189","auth":"","email":"swarm@dev.test","serveraddress":"registry-ice-dev-test.stage1.ng.bluemix.net"}
	auth_str := fmt.Sprintf("{\"username\":\"%s\",\"password\":\"%s\",\"auth\":\"\",\"email\":\"swarm@dev.test\",\"serveraddress\":\"%s\"}", user, psswd, reg)
	auth_bytes := []byte(auth_str)
	tok := base64.StdEncoding.EncodeToString(auth_bytes)

	glog.Infof("InjectRegAuthHeader:  auth_str=%s  tok=%s", auth_str, tok)
	r.Header.Set("X-Registry-Auth", tok)
}

func GetRegistryApiHosts() (hosts []string) {
	// get service host from Consul
	service := "registry-api"
	hosts = conf.GetServiceHosts(service)
	if len(hosts) == 0 {
		glog.Errorf("Failed to get Registry API host  service=%s", service)
		return
	}
	for _, v := range hosts {
		glog.Infof("Found Registry api host=%s", v)
	}
	return
}

func AddCredsHeaders(req *http.Request, creds auth.Creds) {
	req.Header.Add("namespace", creds.Reg_namespace)
	req.Header.Add("apikey", creds.Apikey)
	req.Header.Add("orguuid", creds.Orguuid)
	req.Header.Add("spaceuuid", creds.Space_id)
	req.Header.Add("userid", creds.Userid)

	req.Header.Add("Accept", "application/json")
}

// using internal api exposed by registry microservice
func DoRegistryCall(w http.ResponseWriter, r *http.Request, uriPath string, method string, creds auth.Creds, req_id string) {
	// get service host from Consul
	hosts := GetRegistryApiHosts()
	if len(hosts) == 0 {
		ErrorHandlerWithMsg(w, r, 500, "Failed to get Registry API host")
		return
	}
	//Call service endpoint
	for i := 0; i < len(hosts); i++ {
		url := "http://" + hosts[i] + uriPath
		glog.Infof("Will call Registry... url=%s req_id=%s", url, req_id)
		client := &http.Client{}
		req, _ := http.NewRequest(method, url, nil)
		AddCredsHeaders(req, creds)
		resp, err := client.Do(req)
		if err != nil {
			//try next reg server
			continue
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		glog.Infof("Registry response req_id=%s statusCode=%d body=%s", req_id, resp.StatusCode, httphelper.PrettyJson(body))

		//send response back to client
		w.WriteHeader(resp.StatusCode)
		io.WriteString(w, string(body))
		return
	}
	glog.Warningf("None of the Registry servers responded req_id=%s", req_id)
	return
}

// call internal registry api server to get image metadata
func invoke_reg_inspect(w http.ResponseWriter, r *http.Request, img string, creds auth.Creds, req_id string) {
	uriPath := "/v1/imageJson?imageName=" + img
	method := r.Method
	DoRegistryCall(w, r, uriPath, method, creds, req_id)
}

//implement image list by invoking search api of Containers registry
//return json to docker cli
//DockerHandler will print req processing completion message and exit right after this method
func invoke_reg_list(w http.ResponseWriter, r *http.Request, creds auth.Creds, req_id string) {
	//Not recommended approach. These urls go to registry.ng.bluemix.net
	//ns_url := "http://" + conf.GetRegLocation() + "/v1/namespaces/" + namespace
	//lib_url := "http://" + conf.GetRegLocation() + "/v1/namespaces/library"

	uriPath := "/v1/imageList/" + creds.Reg_namespace
	method := r.Method
	DoRegistryCall(w, r, uriPath, method, creds, req_id)
}

//TODO: reimplement using micrservice when available
func invoke_reg_rmi(w http.ResponseWriter, r *http.Request, img string, creds auth.Creds, req_id string) {
	//img valid format:  host/namespace/name:tag
	sl := strings.Split(img, "/")
	if len(sl) != 3 {
		glog.Errorf("Not valid image name  img=%s req_id=%s", img, req_id)
		ErrorHandlerWithMsg(w, r, 500, "Not valid image name")
		return
	}
	reg := sl[0]
	namespace := sl[1]
	img_name_and_tag := strings.Split(sl[2], ":")
	if len(img_name_and_tag) != 2 {
		glog.Errorf("Not valid image name  img=%s req_id=%s", img, req_id)
		ErrorHandlerWithMsg(w, r, 500, "Not valid image name")
		return
	}
	img_name := img_name_and_tag[0]
	img_tag := img_name_and_tag[1]

	// Construct the request to registry.
	url := fmt.Sprintf("https://%s/v1/repositories/%s/%s/tags/%s", reg, namespace, img_name, img_tag)
	glog.Infof("Will call Registry... url=%s req_id=%s", url, req_id)
	client := &http.Client{}
	req, _ := http.NewRequest("DELETE", url, nil)
	//TODO: when rmi is implemented by reg microservice 	AddCredsHeaders(req, creds)
	req.Header.Add("Accept", "application/json")

	if _REG_ADMIN_CREDS_ {
		req.SetBasicAuth("admin", conf.GetRegAdminPsswd())
	} else {
		req.SetBasicAuth("apikey", creds.Apikey)
	}

	resp, _ := client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	glog.Infof("Registry status code: %d", resp.StatusCode)
	glog.Infof("Registry response: %s", httphelper.PrettyJson(body))

	//send response back to client
	body_str := ""
	if resp.StatusCode == 200 {
		body_str = fmt.Sprintf("[{\"Deleted\":\"%s\"}]", img)
	} else {
		body_str = string(body)
	}
	w.WriteHeader(resp.StatusCode)
	io.WriteString(w, body_str)
	return
}

//////////////////////////// image names extraction and validation ops

func is_img_valid(img string, namespace string) bool {
	if img == "" {
		return true
	}

	// img general format is reg_host/namesapce/imgname:tag or reg_host/imgname:tag
	sl := strings.Split(img, "/")
	if len(sl) <= 2 {
		// public/lib image
		return true
	}

	// we have an img with a namespace
	if !strings.Contains(sl[0], ".bluemix.net") {
		//Dckerhub or other reg image --> OK
		return true
	}

	// we have a Containers reg img with namespace
	// limit access only to this environment's registry
	if namespace == sl[1] && conf.GetRegLocation() == sl[0] {
		return true
	}
	return false
}

func get_image_from_container_create(body []byte) (img string) {
	// look for "Image":"..."
	var f interface{}
	err := json.Unmarshal(body, &f)
	if err != nil {
		glog.Errorf("get_image_from_container_create: error in json unmarshalling, err=%v", err)
		return
	}
	m := f.(map[string]interface{})
	for k, v := range m {
		if k == "Image" {
			img = v.(string)
			glog.Infof("get_image_from_container_create: found img=%s", img)
			return
		}
	}
	glog.Warning("get_image_from_container_create: did not find Image in json body")
	return
}

func getImageFullnameFromVars(vars map[string]string) (fullname string) {
	if _, ok := vars["{img}"]; !ok {
		return
	}
	fullname = vars["{img}"]
	if ns, ok := vars["{ns}"]; ok {
		fullname = ns + "/" + fullname
	}
	if reg, ok := vars["{reg}"]; ok {
		fullname = reg + "/" + fullname
	}
	return
}

func get_image_from_image_create(reqUri string) (img string) {
	//look for ?fromImage=...&registry=...
	u, err := url.Parse(reqUri)
	if err != nil {
		glog.Error(err)
		return
	}
	q := u.Query() // q is map[string][]string
	fromImage := q.Get("fromImage")
	registry := q.Get("registry)")
	if registry == "" {
		img = fromImage
		return
	}
	if strings.Contains(fromImage, registry) {
		img = fromImage
		return
	}
	img = registry + "/" + fromImage
	return
}
