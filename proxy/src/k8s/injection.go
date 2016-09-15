package k8s

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
)

func (kind Kind) Inject(keyValue KeyValue, path ...string) ([]byte, error) {
	err := inject(keyValue, kind.data, path)
	if err != nil {
		return nil, err
	}

	return json.Marshal(kind.data)
}

// Recursively inject the KeyValue
func inject(keyValue KeyValue, data map[string]interface{}, path []string) error {
	if len(path) == 0 {
		if data[keyValue.Key] == "" || data[keyValue.Key] == nil {
			glog.Infof("Injecting: %v with value: %v", keyValue.Key, keyValue.Value)
			data[keyValue.Key] = keyValue.Value

			return nil
		} else {
			glog.Infof("%v already exists: %v", keyValue.Key, data[keyValue.Key])

			return errors.New(fmt.Sprintf("Illegal usage of %v", keyValue.Key))
		}
	} else if data[path[0]] == nil {
		if len(path) == 1 {
			data[path[0]] = make(map[string]interface{})
		} else {
			return errors.New(fmt.Sprintf("%v not found", path[0]))
		}
	}

	return inject(keyValue, data[path[0]].(map[string]interface{}), path[1:])
}
