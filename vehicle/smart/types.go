package smart

import (
	"strconv"
	"time"
)

type StatusResponse struct {
	PreCond   struct{}
	ChargeOpt struct{}
	Status    struct {
		StatusData struct {
			Odo           Value
			RangeElectric Value
			Soc           Value
		} `json:"data"`
	}
	Images           []string
	Error            string
	ErrorDescription string `json:"error_description"`
}

type Value struct {
	Status int
	Value  float64
	Ts     TimeSecs
}

// TimeSecs implements JSON unmarshal for Unix timestamps in seconds
type TimeSecs struct {
	time.Time
}

// UnmarshalJSON decodes unix timestamps in ms into time.Time
func (ct *TimeSecs) UnmarshalJSON(data []byte) error {
	i, err := strconv.ParseInt(string(data), 10, 64)

	if err == nil {
		t := time.Unix(i, 0)
		(*ct).Time = t
	}

	return err
}
