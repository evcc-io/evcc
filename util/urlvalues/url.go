// Package urlvalues provides functions for working with url.Values
package urlvalues

import (
	"errors"
	"net/url"
	"strings"
)

// Copy creates a deep copy of url values
func Copy(q url.Values) url.Values {
	res := make(url.Values, len(q))
	for k, v := range q {
		res[k] = append([]string{}, v...)
	}
	return res
}

// Merge copies multiple from url values into to
func Merge(to url.Values, from ...url.Values) {
	for _, vv := range from {
		for k, v := range vv {
			to[k] = append(to[k], v...)
		}
	}
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
