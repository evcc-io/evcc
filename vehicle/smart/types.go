package smart

import (
	"strconv"
	"time"
)

type StatusResponse struct {
	PreCond struct {
		Data struct {
			ChargingPower  FloatValue
			ChargingActive BoolValue
			ChargingStatus IntValue
		} `json:"data"`
	}
	ChargeOpt struct{}
	Status    struct {
		Data struct {
			Odo           FloatValue
			RangeElectric FloatValue
			Soc           FloatValue
		} `json:"data"`
	}
	Images           []string
	Error            string
	ErrorDescription string `json:"error_description"`
}

type BoolValue struct {
	Status int
	Value  bool
	Ts     TimeSecs
}

type IntValue struct {
	Status int
	Value  int
	Ts     TimeSecs
}

type FloatValue struct {
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
