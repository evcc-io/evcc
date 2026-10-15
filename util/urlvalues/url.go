// Package urlvalues provides functions for working with url.Values
package urlvalues

import (
	"errors"
	"net/url"
	"strings"
)

// Require verifies that url contains the required non-nil values
func Require(q url.Values, keys ...string) error {
	for _, k := range keys {
		if strings.TrimSpace(q.Get(k)) == "" {
			return errors.New("missing " + k)
		}
	}

	return nil
}
