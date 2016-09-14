package k8s

import (
	"encoding/json"
	"errors"
)

type KeyValue struct {
	Key   string
	Value interface{}
}

// https://github.com/kubernetes/kubernetes/blob/release-1.3/docs/devel/api-conventions.md#types-kinds
// Kinds are objects, lists or "simple"

type Kind struct {
	data map[string]interface{}
}

func KindFromJSON(body []byte) (*Kind, error) {
	data := map[string]interface{}{}
	json.Unmarshal(body, &data)
	if data["kind"] == nil || data["kind"] == "" {
		return nil, errors.New("Not a k8s Kind")
	}

	return &Kind{data: data}, nil
}

func (k Kind) GetType() string {
	return k.data["kind"].(string)
}

func (k Kind) Is(kinds ...string) bool {
	for _, kind := range kinds {
		if k.data["kind"] == kind {
			return true
		}
	}

	return false
}
