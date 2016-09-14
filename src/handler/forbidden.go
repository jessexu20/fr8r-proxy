package handler

import (
	"io"
	"net/http"

	"github.com/golang/glog"
)

//Return 403 for Forbidden URIs
func ForbiddenOperationHandler(w http.ResponseWriter, r *http.Request, msg string) {
	glog.Warningf("ForbiddenOperationHandler triggered, URI=%s, returning error 403, msg=%s", r.RequestURI, msg)
	w.WriteHeader(403)
	io.WriteString(w, msg+"\n")
}
