package jq

import (
	"encoding/json"

	"github.com/itchyny/gojq"
	"github.com/pkg/errors"
)

// Query executes a compiled jq query against given input. It expects a single result only.
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
		return nil, errors.Wrap(err, "jq: query failed")
	}

	if _, ok := iter.Next(); ok {
		return nil, errors.New("jq: too many results")
	}

	return v, nil
}
