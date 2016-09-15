package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/golang/glog"
)

//return general status code
func OkHandler(w http.ResponseWriter, r *http.Request, status int) {
	glog.Infof("OkHandler triggered, URI=%s, returning status %d", r.RequestURI, status)
	w.WriteHeader(status)
	//s := fmt.Sprintf("%d success while processing request!\n", status)
	//io.WriteString(w, s)
}

func OkHandlerWithMsg(w http.ResponseWriter, r *http.Request, status int, msg string) {
	glog.Infof("OkHandlerWithMsg triggered, URI=%s, status=%d, msg=%s", r.RequestURI, status, msg)
	w.WriteHeader(status)
	s := fmt.Sprintf("%d %s!\n", status, msg)
	io.WriteString(w, s)
}
