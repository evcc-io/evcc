package warp

import (
	"encoding/json"
	"math"
)

func (f *FloatWithNaN) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*f = FloatWithNaN(math.NaN())
		return nil
	}
	return json.Unmarshal(b, (*float64)(f))
}

func (f FloatWithNaN) IsNaN() bool {
	return math.IsNaN((float64)(f))
}
