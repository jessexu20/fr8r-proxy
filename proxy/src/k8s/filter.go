package k8s

import (
	"encoding/json"
	"github.com/golang/glog"
	"regexp"
	"strings"
)

type filterObject struct {
	filterType string
	selector   Selector
	field      string
	value      interface{}
	regex      string
}

type FilterCollection struct {
	filters map[string][]filterObject
}

type Selector struct {
	Path []string
	Test func(interface{}) bool
}

func NotNilSelector(path ...string) *Selector {
	return &Selector{
		Path: path,
		Test: func(test interface{}) bool {
			return test != nil
		},
	}
}

func IsEqualsSelector(equals interface{}, path ...string) *Selector {
	return &Selector{
		Path: path,
		Test: func(test interface{}) bool {
			return test == equals
		},
	}
}

func NewFilterCollection() *FilterCollection {
	return &FilterCollection{filters: make(map[string][]filterObject)}
}

func (collection FilterCollection) addFilter(kind string, filter filterObject) {
	filters := collection.filters[kind]
	if filters == nil {
		filters = make([]filterObject, 1)
		filters[0] = filter
	} else {
		filters = append(filters, filter)
	}

	collection.filters[kind] = filters
}

func (collection FilterCollection) AddRemoveFilter(kind string, selector *Selector, field string) {
	filter := filterObject{filterType: "remove", selector: *selector, field: field}

	collection.addFilter(kind, filter)
}

func (collection FilterCollection) AddReplaceFilter(value interface{}, kind string, selector *Selector, field string) {
	filter := filterObject{filterType: "replace", selector: *selector, value: value, field: field}

	collection.addFilter(kind, filter)
}

func (collection FilterCollection) AddEmptyFilter(kind string, selector *Selector, field string) {
	filter := filterObject{filterType: "replace", selector: *selector, value: "", field: field}

	collection.addFilter(kind, filter)
}

func (collection FilterCollection) AddRegexFilter(regex string, value string, kind string, selector *Selector, field string) {
	filter := filterObject{filterType: "regex", selector: *selector, value: value, field: field, regex: regex}

	collection.addFilter(kind, filter)
}

func (collection FilterCollection) ApplyToJSON(body []byte) ([]byte, bool) {
	// Get the Kind object, fails gracefully
	kind, err := KindFromJSON(body)
	if err != nil {
		glog.Warningf("%v", err)
		return body, false
	}

	// Apply filters if necessary
	if !collection.filter(kind.data) {
		// No filters applied, don't marshal the JSON and returns the unchanged body
		return body, false
	}

	// Return the filtered JSON, fails gracefully
	filteredBody, err := json.Marshal(kind.data)
	if err != nil {
		glog.Warningf("%v", err)
		return body, false
	}

	return filteredBody, true
}

func (collection FilterCollection) filter(data map[string]interface{}) bool {
	// Check if we have a valid Kind object
	kindType, ok := data["kind"].(string)
	if !ok {
		return false
	}

	// If it is a list filter the items.
	// A List has the suffix List (e.g. List, PodList, ServiceList, NodeList, etc.).
	// At the moment we can't apply filters to lists, only yo list items
	if strings.HasSuffix(kindType, "List") {
		var isGenericList = kindType == "List"

		var itemsFilters = []filterObject{}
		if !isGenericList {
			// Check if we have filters for the expected list item type
			itemsType := strings.TrimSuffix(kindType, "List")
			itemsFilters = collection.filters[itemsType]
			if itemsFilters == nil || len(itemsFilters) == 0 {
				return false
			}
		}

		// Check if we have items and items is an array
		items, ok := data["items"].([]interface{})
		if !ok {
			return false
		}

		// Iterate over all the items
		var filtered = false
		for _, item := range items {
			// Check if an item is a JSON object
			data, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			// Generic lists can have any kind of objects
			if isGenericList {
				// Filter the object recursively
				if collection.filter(data) {
					filtered = true
				}
			} else {
				// Filter the object
				if filterData(data, itemsFilters) {
					filtered = true
				}
			}
		}

		return filtered
	} else {
		// If it isn't a list apply the filters (if any)
		filters := collection.filters[kindType]
		if filters != nil && len(filters) > 0 {
			return filterData(data, filters)
		} else {
			// We have no filters, just return
			return false
		}
	}
}

func filterData(data map[string]interface{}, filters []filterObject) bool {
	var filtered = false
	for _, filter := range filters {
		if filter.apply(data) {
			filtered = true
		}
	}

	return filtered
}

func (filter filterObject) apply(data map[string]interface{}) bool {
	if filter.field == "" {
		return false
	}

	parent := findParent(data, filter.selector)
	if parent == nil {
		return false
	}

	switch filter.filterType {

	case "remove":
		return remove(parent, filter.field)

	case "replace":
		return replace(parent, filter.value, filter.field)

	case "regex":
		return replaceRegex(parent, filter.regex, filter.value, filter.field)

	default:
		return false

	}
}

func findParent(data map[string]interface{}, selector Selector) map[string]interface{} {
	if selector.Path == nil || selector.Test == nil || len(selector.Path) == 0 {
		return nil
	}

	var object = data
	for _, pathComponent := range selector.Path[:len(selector.Path)-1] {
		subObject, ok := object[pathComponent].(map[string]interface{})
		if !ok {
			return nil
		}

		object = subObject
	}

	if selector.Test(object[selector.Path[len(selector.Path)-1]]) {
		return object
	}

	return nil
}

func remove(data map[string]interface{}, field string) bool {
	if data[field] == nil {
		return false
	}

	delete(data, field)
	return true
}

func replace(data map[string]interface{}, value interface{}, field string) bool {
	if data[field] == nil || data[field] == value {
		return false
	}

	data[field] = value
	return true
}

func replaceRegex(data map[string]interface{}, regex string, value interface{}, field string) bool {
	if data[field] == nil || regex == "" {
		return false
	}

	v, ok := value.(string)
	if !ok {
		return false
	}

	s, ok := data[field].(string)
	if !ok {
		return false
	}

	r, err := regexp.Compile(regex)
	if err != nil {
		return false
	}

	data[field] = r.ReplaceAllString(s, v)
	return true
}
