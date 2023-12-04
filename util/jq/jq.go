package jq

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/itchyny/gojq"
)

// Query executes a compiled jq query against given json. It expects a single result only.
func Query(query *gojq.Query, input []byte) (interface{}, error) {
	var j interface{}
	if err := json.Unmarshal(input, &j); err != nil {
		return j, err
	}

	iter := query.Run(j)

	v, ok := iter.Next()
	if !ok {
		return nil, errors.New("jq: empty result")
	}

	if err, ok := v.(error); ok {
		return nil, fmt.Errorf("jq: query failed: %v", err)
	}

	if _, ok := iter.Next(); ok {
		return nil, errors.New("jq: too many results")
	}

	return v, nil
}

// Float64 converts interface to float64
func Float64(v interface{}) (float64, error) {
	switch v := v.(type) {
	case int:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected float type: %T", v)
	}
}

// Int64 converts interface to int64
func Int64(v interface{}) (int64, error) {
	switch v := v.(type) {
	case int:
		return int64(v), nil
	case float64:
		if float64(int64(v)) == v {
			return int64(v), nil
		}
		return 0, fmt.Errorf("unexpected int64: %v", v)
	default:
		return 0, fmt.Errorf("unexpected int64 type: %T", v)
	}
}

// String converts interface to string
func String(v interface{}) (string, error) {
	switch v := v.(type) {
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("unexpected string type: %T", v)
	}
}

// Bool converts interface to bool
func Bool(v interface{}) (bool, error) {
	switch v := v.(type) {
	case bool:
		return v, nil
	default:
		return false, fmt.Errorf("unexpected bool type: %T", v)
	}
}
