package ngeso

import (
	"strings"
	"time"
)

// shortRFC3339Timestamp is a custom JSON encoder / decoder for shortened RFC3339-compliant timestamps (those without seconds).
// Please don't ask why NGESO uses this format instead of standards-compliant RFC3339. 🇬🇧
type shortRFC3339Timestamp struct {
	time.Time
}

const s3339Layout = "2006-01-02T15:04Z"

func (ct *shortRFC3339Timestamp) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		ct.Time = time.Time{}
		return
	}
	ct.Time, err = time.Parse(s3339Layout, s)
	return
}

func (ct *shortRFC3339Timestamp) MarshalJSON() ([]byte, error) {
	if ct.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(ct.Time.Format(s3339Layout)), nil
}

func (ct *shortRFC3339Timestamp) IsSet() bool {
	return !ct.IsZero()
}
