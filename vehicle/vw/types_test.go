package vw

import (
	"encoding/json"
	"math"
	"testing"
)

func TestTemp(t *testing.T) {
	data := `[{
		"content": 2930,
		"timestamp": "time"
	}, {
		"content": "invalid",
		"timestamp": "time"
	}]`

	var temps []TimedTemperature
	if err := json.Unmarshal([]byte(data), &temps); err != nil {
		t.Error(err)
	} else {
		if v := temps[0].Content; v != 20 {
			t.Error("invalid temp 0", v)
		}
		if v := temps[1].Content; !math.IsNaN(v) {
			t.Error("invalid temp 1", v)
		}
	}
}
