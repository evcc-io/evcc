package greenely

import (
	"strings"
	"time"
)

// localTime is a wrapper around time.Time to handle the "YYYY-MM-DD HH:MM" format from the API.
type localTime struct {
	time.Time
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (ct *localTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		return nil
	}

	t, err := time.ParseInLocation("2006-01-02 15:04", s, time.Local)
	if err != nil {
		return err
	}
	ct.Time = t
	return nil
}

type SpotPrice struct {
	Data map[int]struct {
		Price          int
		LocalTime      localTime `json:"localtime"`
		QuartersPrices struct {
			Quarter1 int
			Quarter2 int
			Quarter3 int
			Quarter4 int
		} `json:"quarters_prices"`
	}
}
