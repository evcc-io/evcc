package psa

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/dylanmei/iso8601"
)

// Duration is a time.Duration that can be unmarshalled from JSON
type Duration struct {
	time.Duration
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil

	case string:
		var err error
		d.Duration, err = iso8601.ParseDuration(value)
		return err

	default:
		return errors.New("invalid duration")
	}
}
