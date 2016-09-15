package conf

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/golang/glog"
)

type ConsulService struct {
	Node        string
	Address     string
	ServiceID   string
	ServiceName string
	ServiceTags []string
	ServicePort int
}

func GetServiceHosts(service string) (hosts []string) {
	consul_host := GetConsulIp() + ":" + strconv.Itoa(GetConsulPort())
	consul_url := "http://" + consul_host + "/v1/catalog/service/" + service
	glog.Infof("Will invoke Consul... url=%s", consul_url)
	//invoke Consul to get service metadata
	client := &http.Client{}
	req, err := http.NewRequest("GET", consul_url, nil)
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("Error connecting to Consul: %v", err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	glog.Infof("Consul raw response: %s", string(body))
	if resp.StatusCode != 200 {
		return
	}

	//parse body json
	var consulService []ConsulService
	err = json.Unmarshal(body, &consulService)
	if err != nil {
		glog.Errorf("Error in parsing Consul response: error=%v", err)
		return
	}

	//return all endpoints
	hosts = make([]string, len(consulService))
	for k, v := range consulService {
		hosts[k] = v.Address + ":" + strconv.Itoa(v.ServicePort)
	}
	return
}
