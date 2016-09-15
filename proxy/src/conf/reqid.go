package conf

import (
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"sync"

	"github.com/golang/glog"
)

var glob_req_id = 0
var glob_req_id_mutex sync.Mutex

func GetReqId() string {
	//inc counter anyway
	glob_req_id_mutex.Lock()
	glob_req_id += 1 //this op should be in a critical section
	surr := glob_req_id
	glob_req_id_mutex.Unlock()
	if IsSurrogateIds() {
		return strconv.Itoa(surr)
	}
	// generate random string id, needed in case of horizontal scaling
	b := make([]byte, 10)
	_, err := rand.Read(b)
	if err != nil {
		glog.Error("error in rand num generator:", err)
		return strconv.Itoa(surr) //"0"
	}
	// The slice should now contain random bytes instead of only zeroes.
	req_id := base64.StdEncoding.EncodeToString(b)
	return req_id
}

// num of requests served by this instance
func GetNumServedRequests() int {
	return glob_req_id
}
