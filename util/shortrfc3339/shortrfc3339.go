// Package shortrfc3339 implements helpers for working with shortened RFC-3339 compliant timestamps (those without seconds).
package shortrfc3339

import (
	"strings"
	"time"
)

// ShortRFC3339Timestamp is a custom JSON encoder / decoder for shortened RFC3339-compliant timestamps (those without seconds).
type ShortRFC3339Timestamp struct {
	time.Time
}

// Layout is the time.Parse compliant parsing string for use when parsing Shortened RFC-3339 compliant timestamps.
const Layout = "2006-01-02T15:04Z"

func (ct *ShortRFC3339Timestamp) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		ct.Time = time.Time{}
		return
	}
	ct.Time, err = time.Parse(Layout, s)
	return
}

func (ct *ShortRFC3339Timestamp) MarshalJSON() ([]byte, error) {
	if ct.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(ct.Time.Format(Layout)), nil
}
