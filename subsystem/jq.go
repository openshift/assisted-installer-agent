package subsystem

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/onsi/gomega/types"
)

// JQ runs the given `jq` filter on the given object and returns the list of results. The returned
// slice will never be nil; if there are no results it will be empty.
func JQ(filter string, input interface{}) (results []interface{}, err error) {
	// Parse the filter:
	query, err := gojq.Parse(filter)
	if err != nil {
		return
	}

	// If the input is an array of bytes or an string then we need to unmarshal it, so that the
	// rest of the code doesn't need to handle that explicitly:
	switch typed := input.(type) {
	case []byte:
		err = json.Unmarshal(typed, &input)
		if err != nil {
			return
		}
	case string:
		err = json.Unmarshal([]byte(typed), &input)
		if err != nil {
			return
		}
	}

	// Run the query:
	iterator := query.Run(input)
	for {
		result, ok := iterator.Next()
		if !ok {
			break
		}
		results = append(results, result)
	}
	return
}

// MatchJQ creates a matcher that checks that the all the results of applying a `jq` filter to the
// actual value is the given expected value.
func MatchJQ(filter string, expected interface{}) types.GomegaMatcher {
	return &jqMatcher{
		filter:   filter,
		expected: expected,
	}
}

type jqMatcher struct {
	filter   string
	expected interface{}
	results  []interface{}
}

func (m *jqMatcher) Match(actual interface{}) (success bool, err error) {
	// Run the query:
	m.results, err = JQ(m.filter, actual)
	if err != nil {
		return
	}

	// Check that there is at least one result:
	if len(m.results) == 0 {
		return
	}

	// We consider the match sucessful if all the results returned by the JQ filter are exactly
	// equal to the expected value.
	success = true
	for _, result := range m.results {
		if !reflect.DeepEqual(result, m.expected) {
			success = false
			break
		}
	}
	return
}

func (m *jqMatcher) FailureMessage(actual interface{}) string {
	return fmt.Sprintf(
		"Expected results of running 'jq' filter\n\t%s\n"+
			"on input\n\t%s\n"+
			"to be\n\t%s\n"+
			"but at list one of the following results isn't\n\t%s\n",
		m.filter, m.pretty(actual), m.pretty(m.expected), m.pretty(m.results),
	)
}

func (m *jqMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf(
		"Expected results of running 'jq' filter\n\t%s\n"+
			"on input\n\t%s\n"+
			"to not be\n\t%s\n",
		m.filter, m.pretty(actual), m.pretty(m.expected),
	)
}

func (m *jqMatcher) pretty(object interface{}) string {
	// If the object is an array of bytes or an string then we need to unmarshal it so that we
	// can later marshal it with indentation:
	switch typed := object.(type) {
	case []byte:
		var tmp interface{}
		if json.Unmarshal(typed, &tmp) == nil {
			object = tmp
		}
	case string:
		var tmp interface{}
		if json.Unmarshal([]byte(typed), &tmp) == nil {
			object = tmp
		}
	}

	// Marshal the object with indentation, to make it easier to read:
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("\t", "  ")
	err := encoder.Encode(object)
	if err != nil {
		return fmt.Sprintf("\t%v", object)
	}
	return strings.TrimRight(buffer.String(), "\n")
}
