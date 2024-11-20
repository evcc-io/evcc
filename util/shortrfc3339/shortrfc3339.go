// Package shortrfc3339 implements helpers for working with shortened RFC-3339 compliant timestamps (those without seconds).
package shortrfc3339

import (
	"encoding/xml"
	"strings"
	"time"
)

// Timestamp is a custom JSON encoder / decoder for shortened RFC3339-compliant timestamps (those without seconds).
type Timestamp struct {
	time.Time
}

// Layout is the time.Parse compliant parsing string for use when parsing Shortened RFC-3339 compliant timestamps.
const Layout = "2006-01-02T15:04Z"

func (ct *Timestamp) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		ct.Time = time.Time{}
		return
	}
	ct.Time, err = time.Parse(Layout, s)
	return
}

func (ct *Timestamp) MarshalJSON() ([]byte, error) {
	if ct.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(ct.Time.Format(Layout)), nil
}

func (ct *Timestamp) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	err := d.DecodeElement(&s, &start)
	if err == nil {
		ct.Time, err = time.Parse(Layout, s)
	}
	return err
}
