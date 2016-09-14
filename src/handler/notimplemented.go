package handler

import (
	"io"
	"net/http"

	"github.com/golang/glog"
)

//Return 501 for non-supported URIs
func NotImplementedHandler(w http.ResponseWriter, r *http.Request) {
	glog.Errorf("NotImplementedHandler triggered, URI=%s, returning error 501", r.RequestURI)
	w.WriteHeader(501)
	io.WriteString(w, "Not implemented\n")
}
