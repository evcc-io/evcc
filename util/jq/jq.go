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
