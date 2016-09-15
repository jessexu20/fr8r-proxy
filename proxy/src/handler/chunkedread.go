package handler

import (
	"io"
	"net/http"

	"httphelper"

	"github.com/golang/glog"
)

func chunkedRWLoop(resp *http.Response, w http.ResponseWriter, req_id string) {
	buffer := make([]byte, 8096)
	var err error
	var nr int
	//bufio_w := bufio.NewWriter(w)

	f, ok := w.(http.Flusher)
	if !ok {
		f = nil
	}

	glog.Info("Response body:")
	tot_nr := 0
	for err == nil {
		//nr, err = io.ReadFull(resp.Body, buffer)  // does not return until buffer is full or err happens
		nr, err = resp.Body.Read(buffer) // returns when reader has some data, big buffer is ok
		if nr > 0 {
			tot_nr += nr
			glog.Infof("len=%d tot_len=%d req_id=%s %s", nr, tot_nr, req_id, httphelper.PrettyJson(buffer[:nr]))
			nw, werr := io.WriteString(w, string(buffer[:nr])) // w.Write(buffer[:nr]) //fmt.Fprintf(bufio_w,"%s", buffer[:nr]) //io.WriteString(w, str)
			if werr != nil || nr != nw {
				glog.Errorf("Error writing data  req_id=%s \n", req_id)
				return
			}
			if f != nil {
				f.Flush()
			}
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			glog.Errorf("received EOF  req_id=%s", req_id)
			return
		}
		if err != nil {
			glog.Errorf("Error reading data  req_id=%s err=%v\n", req_id, err)
			return
		}
	}
}
