// Package urlvalues provides functions for working with url.Values
package urlvalues

import (
	"errors"
	"net/url"
	"slices"
	"strings"
)

// Copy creates a deep copy of url values
func Copy(q url.Values) url.Values {
	res := make(url.Values, len(q))
	for k, v := range q {
		res[k] = slices.Clone(v)
	}
	return res
}

// Require verifies that url contains the required non-nil values
func Require(q url.Values, keys ...string) error {
	for _, k := range keys {
		if strings.TrimSpace(q.Get(k)) == "" {
			return errors.New("missing " + k)
		}
	}

	return nil
}
