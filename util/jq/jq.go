package jq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/itchyny/gojq"
)

// Query executes a compiled jq query against given json. It expects a single result only.
func Query(query *gojq.Query, input []byte) (any, error) {
	return QueryContext(context.Background(), query, input)
}

// QueryContext executes a compiled jq query against given json, aborting when ctx is done.
// It expects a single result only. Use this instead of Query whenever the query originates
// from an untrusted source, since jq is turing-complete and evaluation may not terminate.
func QueryContext(ctx context.Context, query *gojq.Query, input []byte) (any, error) {
	var j any
	if err := json.Unmarshal(input, &j); err != nil {
		return j, err
	}

	iter := query.RunWithContext(ctx, j)

	v, ok := iter.Next()
	if !ok {
		return nil, errors.New("jq: empty result")
	}

	if err, ok := v.(error); ok {
		// halt/halt_error do not terminate the iterator by themselves
		var he *gojq.HaltError
		if errors.As(err, &he) {
			return nil, errors.New("jq: query halted")
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("jq: %w", ctxErr)
		}

		return nil, fmt.Errorf("jq: query failed: %v", err)
	}

	if _, ok := iter.Next(); ok {
		return nil, errors.New("jq: too many results")
	}

	return v, nil
}
