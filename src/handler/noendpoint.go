package handler

import (
	"io"
	"net/http"

	"github.com/golang/glog"
)

//Return 404 for non-supported URIs
func NoEndpointHandler(w http.ResponseWriter, r *http.Request) {
	glog.Errorf("NoEndpointHandler triggered, URI=%s, returning error 404", r.RequestURI)
	//http.NotFound(w, r)
	w.WriteHeader(404)
	io.WriteString(w, "404 not found!\n")
}
