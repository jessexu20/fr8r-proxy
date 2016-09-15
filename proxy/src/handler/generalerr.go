package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/golang/glog"
)

//return general status code
func ErrorHandler(w http.ResponseWriter, r *http.Request, status int) {
	glog.Errorf("ErrorHandler triggered, URI=%s, returning error %d", r.RequestURI, status)
	w.WriteHeader(status)
	s := fmt.Sprintf("%d error encountered while processing request!\n", status)
	io.WriteString(w, s)
}

func ErrorHandlerWithMsg(w http.ResponseWriter, r *http.Request, status int, msg string) {
	glog.Errorf("ErrorHandler triggered, URI=%s, error=%d, msg=%s", r.RequestURI, status, msg)
	w.WriteHeader(status)
	s := fmt.Sprintf("%d %s!\n", status, msg)
	io.WriteString(w, s)
}
