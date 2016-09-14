package httphelper

import (
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/golang/glog"
	"io"
)

func InitProxyHijack(w http.ResponseWriter, cc *httputil.ClientConn, req_id string, proto string) {
	if proto != "TCP" {
		glog.Errorf("hijack protocol %s not supported\n", proto)
		return
	}

	//hijack client conn (act as server on this conn)
	cli_conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		glog.Errorf("httproxy hijacking error\n")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cli_conn.Close()

	cli_conn.Write([]byte{})
	//cli_conn.Write([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x74, 0x65, 0x73, 0x74, 0x0a})

	//hijack server conn (act as client on this conn)
	srv_conn, _ := cc.Hijack()
	defer srv_conn.Close()

	glog.Infof("Client: (%s, %s) -> (%s, %s)", cli_conn.LocalAddr().Network(), cli_conn.LocalAddr().String(), cli_conn.RemoteAddr().Network(), cli_conn.RemoteAddr().String())
	glog.Infof("Server: (%s, %s) -> (%s, %s)", srv_conn.LocalAddr().Network(), srv_conn.LocalAddr().String(), srv_conn.RemoteAddr().Network(), srv_conn.RemoteAddr().String())

	tcpHijack(cli_conn, srv_conn, req_id)
	//time.Sleep(100*time.Millisecond) //allow time for go routines to shutdown after hijack completion
}

//implement tcp hijack loop forwarding raw tcp messages between cli and srv
func tcpHijack(client, server io.ReadWriter, req_id string) {
	//start 2 blocking read/forward loops, BUT exit as soon as one of them exits
	var wg sync.WaitGroup

	wg.Add(1) //add 1 only not 2, to proceed when one of the two go routines finishes

	prefix := "> (req id: " + req_id + ")"
	go tcpCopy(client, server, prefix, &wg, false)

	prefix = "< (req id: " + req_id + ")"
	go tcpCopy(server, client, prefix, &wg, true)

	//wait until one go routines finishes... the server read loop routine
	//If the 2nd go routine calls wg.Done() at its exit (not our case) you need to increment wg.Add() so no panic happens
	//the 2 conn closures happen at exit of tcpHijack, if the client routine is still running it will receive read error and exit
	wg.Wait()
	//wg.Add(1) Do not wait for the client. Also the client routine should not call wg.Done() on exit

	//allow time for other go routine to flush any data in pipe before sockets are closed
	//this sleep should not be needed now since we exit only when server closes its socket (i.e., server read routine exits)
	//time.Sleep(100*time.Millisecond)

	prefix = "(req id: " + req_id + ")"
	glog.Infof("%s Hijack exit and connections close\n", prefix)
}

func tcpCopy(source io.Reader, dest io.Writer, print_prefix string, wg *sync.WaitGroup, server_read_loop bool) {
	if server_read_loop {
		// the caller, tcpHijack(), should exit only if the server is done
		defer wg.Done()
	}

	glog.Infof("%s rwloop started\n", print_prefix)
	written, err := io.Copy(dest, source)
	if err != nil {
		glog.Errorf("%s Error writing data: %v\n", print_prefix, err)
	}
	glog.Infof("%s %d bytes written", print_prefix, written)
}
