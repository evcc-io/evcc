package util

import (
	"errors"
	"net/url"
)

// RequireValues verifiies that url contains the required non-nil values
func RequireValues(q url.Values, keys ...string) error {
	for _, k := range keys {
		if q.Get(k) == "" {
			return errors.New("missing " + k)
		}
	}

	return nil
}
