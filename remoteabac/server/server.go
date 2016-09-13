package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"k8s.io/kubernetes/pkg/api/unversioned"
	abacapi "k8s.io/kubernetes/pkg/apis/abac/v1beta1"
	"k8s.io/kubernetes/pkg/apis/authorization/v1beta1"
	"k8s.io/kubernetes/pkg/auth/authorizer"
	"k8s.io/kubernetes/pkg/auth/authorizer/abac"
	"k8s.io/kubernetes/pkg/auth/user"
)

const (
	defaultAddress string = ":8888"
	tmpFile               = "/tmp/abac-policy"
)

type remoteABACServer struct {
	address       string
	policyFile    string
	tlsCertFile   string
	tlsPrivateKey string
	auth          atomic.Value
	kapi          etcd.KeysAPI
	path          string
}

type userRequest struct {
	User       string `json:"user,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	ReqType    string `json:"reqtype"`
	Privileged bool   `json:"privileged,omitempty"`
}

type userResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type userResource struct {
	name     string
	readOnly bool
}

var allowedResources = []userResource{
	{name: "deployments", readOnly: false},
	{name: "events", readOnly: false},
	{name: "pods", readOnly: false},
	{name: "replicationcontrollers", readOnly: false},
	{name: "replicasets", readOnly: false},
	{name: "secrets", readOnly: false},
	{name: "services", readOnly: false},
	{name: "limitranges", readOnly: true},
	{name: "resourcequotas", readOnly: true},
	{name: "jobs", readOnly: false}}

func New() *remoteABACServer {
	return &remoteABACServer{
		address: defaultAddress,
	}
}

// Authorize a web request
func (s *remoteABACServer) authorize(w http.ResponseWriter, r *http.Request) {
	// Decode request data
	var req v1beta1.SubjectAccessReview
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// TODO: is this the right way to handle request data we do not recognize?
		log.Printf("Cannot parse request: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	attribs := authorizer.AttributesRecord{}
	attribs.User = &user.DefaultInfo{Name: req.Spec.User}
	if req.Spec.ResourceAttributes != nil {
		attribs.Verb = req.Spec.ResourceAttributes.Verb
		attribs.Namespace = req.Spec.ResourceAttributes.Namespace
		attribs.APIGroup = req.Spec.ResourceAttributes.Group
		attribs.Resource = req.Spec.ResourceAttributes.Resource
		attribs.ResourceRequest = true
	} else if req.Spec.NonResourceAttributes != nil {
		attribs.Verb = req.Spec.NonResourceAttributes.Verb
		attribs.Path = req.Spec.NonResourceAttributes.Path
		attribs.ResourceRequest = false
	}

	// Check ABAC authorization policy
	auth, ok := s.auth.Load().(authorizer.Authorizer)
	if !ok {
		log.Printf("Cannot convert data to authorizer.Authorizer\n")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	ret := auth.Authorize(attribs)

	// Create response
	res := &v1beta1.SubjectAccessReview{}
	res.Kind = req.Kind
	res.APIVersion = req.APIVersion
	if ret != nil {
		log.Printf("deny access to %s\n", req.Spec.User)
		res.Status.Allowed = false
		res.Status.Reason = ret.Error()
	} else {
		log.Printf("allow access to %s\n", req.Spec.User)
		res.Status.Allowed = true
	}

	// Encode response
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Printf("Cannot encode response: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func genResponse(w http.ResponseWriter, err error, message *string) {
	res := &userResponse{}
	if err == nil {
		res.Status = "OK"
	} else {
		res.Status = err.Error()
	}

	if message != nil {
		res.Message = *message
	}

	// Encode response
	if e := json.NewEncoder(w).Encode(res); e != nil {
		log.Printf("Cannot encode response: %v\n", e)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Supports the following format
//   /users/<user>?privileged=true
//   /users/<user>/<namespace>
func (s *remoteABACServer) addUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	namespace := vars["namespace"]
	privileged := r.URL.Query().Get("privileged")

	if user == "" {
		w.WriteHeader(http.StatusBadRequest)
		genResponse(w, fmt.Errorf("User needs to be specified"), nil)
		return
	}

	if privileged == "" && namespace == "" {
		w.WriteHeader(http.StatusBadRequest)
		genResponse(w, fmt.Errorf("Namespace needs to be specified"), nil)
		return
	}

	log.Printf("add user '%v' to namespace '%v'\n", user, namespace)

	resp, err := s.kapi.Get(context.Background(), s.path, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		genResponse(w, err, nil)
		return
	}

	policy := resp.Node.Value

	serviceAccountExists := false
	serviceAccountName := "system:serviceaccount:" + namespace + ":default"

	// Check if the user already exists
	scanner := bufio.NewScanner(strings.NewReader(policy))
	for scanner.Scan() {
		var po abacapi.Policy
		b := scanner.Bytes()

		// skip comment lines and blank lines
		trimmed := strings.TrimSpace(string(b))
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if err := json.NewDecoder(strings.NewReader(trimmed)).Decode(&po); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			genResponse(w, err, nil)
			return
		}

		if user == po.Spec.User {
			if po.Spec.Namespace == "*" || po.Spec.Namespace == namespace {
				w.WriteHeader(http.StatusConflict)
				genResponse(w, fmt.Errorf("User '%v' already exist", user), nil)
				return
			}
		}

		if po.Spec.User == serviceAccountName {
			serviceAccountExists = true
		}
	}

	// Adding user
	if privileged == "true" {
		// Resource path
		p0 := abacapi.Policy{
			TypeMeta: unversioned.TypeMeta{
				APIVersion: "abac.authorization.kubernetes.io/v1beta1",
				Kind:       "Policy",
			},
			Spec: abacapi.PolicySpec{
				User:      user,
				Namespace: "*",
				Resource:  "*",
				APIGroup:  "*",
			},
		}
		b := new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(p0); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			genResponse(w, err, nil)
			return
		}
		policy = policy + b.String()

		// Non-resource path
		p1 := abacapi.Policy{
			TypeMeta: unversioned.TypeMeta{
				APIVersion: "abac.authorization.kubernetes.io/v1beta1",
				Kind:       "Policy",
			},
			Spec: abacapi.PolicySpec{
				User:            user,
				NonResourcePath: "*",
			},
		}
		b = new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(p1); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			genResponse(w, err, nil)
			return
		}
		policy = policy + b.String()

	} else {
		for _, resource := range allowedResources {
			p0 := abacapi.Policy{
				TypeMeta: unversioned.TypeMeta{
					APIVersion: "abac.authorization.kubernetes.io/v1beta1",
					Kind:       "Policy",
				},
				Spec: abacapi.PolicySpec{
					User:      user,
					Namespace: namespace,
					Resource:  resource.name,
					Readonly:  resource.readOnly,
					APIGroup:  "*",
				},
			}
			b := new(bytes.Buffer)
			if err := json.NewEncoder(b).Encode(p0); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				genResponse(w, err, nil)
				return
			}
			policy = policy + b.String()
		}
	}

	// Write to etcd backend
	_, err = s.kapi.Set(context.Background(), s.path, policy, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		genResponse(w, err, nil)
		return
	}

	log.Printf("serviceAccountExists: %v, serviceAccountName: %v", serviceAccountExists, serviceAccountName)
	// Check if we need to also add authz for service account in this namespace
	if !serviceAccountExists && privileged != "true" {
		p0 := abacapi.Policy{
			TypeMeta: unversioned.TypeMeta{
				APIVersion: "abac.authorization.kubernetes.io/v1beta1",
				Kind:       "Policy",
			},
			Spec: abacapi.PolicySpec{
				User:      serviceAccountName,
				Namespace: "default",
				Resource:  "services",
				Readonly:  true,
				APIGroup:  "*",
			},
		}
		b := new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(p0); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			genResponse(w, err, nil)
			return
		}
		policy = policy + b.String()

		// Write to etcd backend
		_, err = s.kapi.Set(context.Background(), s.path, policy, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			genResponse(w, err, nil)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	genResponse(w, nil, nil)
}

// Supports the following format
//   /users/<user>
//   /users/<user>/<namespace>
func (s *remoteABACServer) deleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	namespace := vars["namespace"]

	if user == "" {
		w.WriteHeader(http.StatusBadRequest)
		genResponse(w, fmt.Errorf("User needs to be specified"), nil)
		return
	}

	log.Printf("delete user '%v' from namespace '%v'\n", user, namespace)

	resp, err := s.kapi.Get(context.Background(), s.path, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		genResponse(w, err, nil)
		return
	}

	policy := resp.Node.Value

	found := false
	var ret string
	scanner := bufio.NewScanner(strings.NewReader(policy))
	for scanner.Scan() {
		var po abacapi.Policy
		b := scanner.Bytes()

		// skip comment lines and blank lines
		trimmed := strings.TrimSpace(string(b))
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			ret = ret + string(b) + "\n"
			continue
		}

		if err := json.NewDecoder(strings.NewReader(trimmed)).Decode(&po); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			genResponse(w, err, nil)
			return
		}

		if user == po.Spec.User {
			if namespace == "" {
				// Unspecified namespace means we should remove all namespaces associated with this user
				found = true
				continue
			} else if namespace == po.Spec.Namespace {
				found = true
				continue
			}
		}
		ret = ret + string(b) + "\n"
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		genResponse(w, fmt.Errorf("User '%v' not found", user), nil)
		return
	}

	// Encode response
	_, err = s.kapi.Set(context.Background(), s.path, ret, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		genResponse(w, err, nil)
		return
	}
	genResponse(w, nil, nil)
}

// Supports the following format
//   /users
//   /users/<user>
//   /users/<user>/<namespace>
func (s *remoteABACServer) getUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	namespace := vars["namespace"]

	resp, err := s.kapi.Get(context.Background(), s.path, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		genResponse(w, err, nil)
		return
	}

	if user == "" {
		w.WriteHeader(http.StatusBadRequest)
		genResponse(w, nil, &resp.Node.Value)
		return
	}

	policy := resp.Node.Value

	var ret string
	scanner := bufio.NewScanner(strings.NewReader(policy))
	for scanner.Scan() {
		var po abacapi.Policy
		b := scanner.Bytes()

		// skip comment lines and blank lines
		trimmed := strings.TrimSpace(string(b))
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if err := json.NewDecoder(strings.NewReader(trimmed)).Decode(&po); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			genResponse(w, err, nil)
			return
		}

		if user == po.Spec.User {
			if namespace == "" {
				ret = ret + string(b) + "\n"
			} else if namespace == po.Spec.Namespace {
				ret = ret + string(b) + "\n"
			}
			continue
		}
	}

	if ret == "" {
		if namespace != "" {
			w.WriteHeader(http.StatusNotFound)
			genResponse(w, fmt.Errorf("No user and namespace (%v, %v) was found", user, namespace), nil)
		} else {
			w.WriteHeader(http.StatusNotFound)
			genResponse(w, fmt.Errorf("No user '%v' was found", user), nil)
		}
	} else {
		genResponse(w, nil, &ret)
	}
}

// Start Remote ABAC server and set up https mux
func (s *remoteABACServer) Run() {
	s.AddFlags(flag.CommandLine)
	flag.Parse()

	if s.tlsCertFile == "" {
		log.Fatalf("TLS certificate file not defined\n")
	}

	if s.tlsPrivateKey == "" {
		log.Fatalf("TLS key file not defined\n")
	}

	if s.policyFile == "" {
		log.Fatalf("ABAC policy file not defined\n")
	}

	s.init()

	log.Printf("Starting server and listening on %s\n", s.address)
	mux := mux.NewRouter()
	mux.HandleFunc("/authorize", s.authorize)
	mux.HandleFunc("/user", s.getUser).Methods("GET")
	mux.HandleFunc("/user/{user}", s.getUser).Methods("GET")
	mux.HandleFunc("/user/{user}/{namespace}", s.getUser).Methods("GET")
	mux.HandleFunc("/user/{user}", s.deleteUser).Methods("DELETE")
	mux.HandleFunc("/user/{user}/{namespace}", s.deleteUser).Methods("DELETE")
	mux.HandleFunc("/user/{user}", s.addUser).Methods("POST", "PUT")
	mux.HandleFunc("/user/{user}/{namespace}", s.addUser).Methods("POST", "PUT")
	http.ListenAndServeTLS(s.address, s.tlsCertFile, s.tlsPrivateKey, mux)
}

// policyFile can be in different formats to specify their storage medium
// In the most common case, where it is stored as a local file, one can specify
//
// --authorization-policy-file=myfile
//
// In the case of etcd backend, one can specify
//
// --authorization-policy-file=etcd@http://10.10.0.1:2379/path/to/policy/file
//
// One can also specify multiple etcd backends, e.g.,
//
// --authorization-policy-file=etcd@http://10.10.0.1:2379/path/to/policy/file,\
//   http://10.10.0.2:2379/path/to/policy/file,\
//   http://10.10.0.3:2379/path/to/policy/file
//
func (s *remoteABACServer) init() {

	arr := strings.Split(s.policyFile, "@")
	if len(arr) == 1 {
		// This is a local file
		auth, err := abac.NewFromFile(s.policyFile)
		if err != nil {
			log.Fatalf("Loading policy file from %s failed: %\n", s.policyFile, err)
		} else {
			log.Printf("Loaded policy file from %s\n", s.policyFile)
			s.auth.Store(auth)
			return
		}
	} else if len(arr) > 2 {
		log.Fatalf("Policy file is not correctly specified: %s\n", s.policyFile)
	}

	storageType := strings.ToLower(arr[0])
	policyFile := arr[1]

	switch storageType {
	// This is an etcd based file
	case "etcd":
		log.Printf("Loading policy file from etcd: %s\n", policyFile)

		serverList := []string{}
		path := ""

		re := regexp.MustCompile(`(http[s]?://[a-zA-Z0-9\.]+:[0-9]+)/(.+)`)
		locations := strings.Split(policyFile, ",")
		for _, location := range locations {
			result := re.FindStringSubmatch(location)
			if result == nil || len(result) != 3 {
				log.Fatalf("etcd location is not recognized: %s\n", location)
			}

			serverList = append(serverList, result[1])
			if path == "" {
				path = result[2]
			} else if path != result[2] {
				log.Fatalf("All etcd path should be the same, %s does not match others\n", result[2])
			}
		}

		path = "/" + path
		s.path = path
		log.Printf("serverList: %s, path: %s\n", serverList, path)

		cfg := etcd.Config{
			Endpoints:               serverList,
			Transport:               etcd.DefaultTransport,
			HeaderTimeoutPerRequest: time.Second,
		}

		client, err := etcd.New(cfg)
		if err != nil {
			log.Fatalf("Failed to create etcd connection: %v\n", err)
		}

		kapi := etcd.NewKeysAPI(client)
		s.kapi = kapi

		resp, err := kapi.Get(context.Background(), path, nil)
		if err != nil {
			log.Fatalf("Cannot GET %s from etcd server: %v\n", path, err)
		}
		//log.Printf("%q key has %q value\n", resp.Node.Key, resp.Node.Value)

		// Load policy file
		ioutil.WriteFile(tmpFile, []byte(resp.Node.Value), 0644)
		auth, err := abac.NewFromFile(tmpFile)
		if err != nil {
			log.Fatalf("Loading policy file from %s failed: %\n", s.policyFile, err)
		} else {
			log.Printf("Loaded policy file from %s\n", s.policyFile)
			s.auth.Store(auth)

			// Listen for etcd file change
			go func() {
				watcher := kapi.Watcher(path, nil)
				for {
					resp, err := watcher.Next(context.Background())
					if err != nil {
						log.Printf("Encountered error while watching for etcd: %v\n", err)
					} else {
						//log.Printf("%q key has %q value\n", resp.Node.Key, resp.Node.Value)
						ioutil.WriteFile(tmpFile, []byte(resp.Node.Value), 0644)
						auth, err := abac.NewFromFile(tmpFile)
						if err != nil {
							log.Printf("Reloading policy file from %s failed: %\n", s.policyFile, err)
						} else {
							log.Printf("Reloading policy file from %s\n", s.policyFile)
							s.auth.Store(auth)
						}
					}
					time.Sleep(time.Second)
				}
			}()
		}
	default:
		log.Fatalf("Storage type %s is not currently supported\n", storageType)
	}
}

func (s *remoteABACServer) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&s.address, "address", s.address, "Address remote ABAC server listens on (ip:port or :port to listen to all interfaces).")
	fs.StringVar(&s.policyFile, "authorization-policy-file", s.policyFile, "Authorization policy file.")
	fs.StringVar(&s.tlsCertFile, "tls-cert-file", s.tlsCertFile, "File containing x509 Certificate for HTTPS.")
	fs.StringVar(&s.tlsPrivateKey, "tls-private-key-file", s.tlsPrivateKey, "File containing x509 private key matching --tls-cert-file.")
}
