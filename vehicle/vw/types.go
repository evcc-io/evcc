package vw

import (
	"encoding/json"
	"math"
	"strconv"
)

// TimedInt is an int value with timestamp
type TimedInt struct {
	Content   int
	Timestamp string
}

// TimedString is a string value with timestamp
type TimedString struct {
	Content   string
	Timestamp string
}

// TimedTemperature is the api temperature with timestamp
type TimedTemperature struct {
	Content   float64
	Timestamp string
}

func (t *TimedTemperature) UnmarshalJSON(data []byte) error {
	var temp struct {
		Content   json.RawMessage // handle "invalid"
		Timestamp string
	}

	err := json.Unmarshal(data, &temp)
	if err == nil {
		(*t).Timestamp = temp.Timestamp

		if val, err := strconv.Atoi(string(temp.Content)); err == nil {
			(*t).Content = temp2Float(val)
		} else {
			(*t).Content = math.NaN()
		}
	}

	return err
}

// temp2Float converts api temp to float value
func temp2Float(i int) float64 {
	return float64(i)/10 - 273
}
