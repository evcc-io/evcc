package homeassistant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToCelsius(t *testing.T) {
	for _, tc := range []struct {
		value float64
		unit  string
		want  float64
		err   bool
	}{
		{21.5, "°C", 21.5, false},
		{21.5, "", 21.5, false},
		{32, "°F", 0, false},
		{212, "°F", 100, false},
		{273.15, "K", 0, false},
		{50, "%", 0, true},
	} {
		got, err := toCelsius(tc.value, tc.unit)
		if tc.err {
			assert.Error(t, err, tc.unit)
			continue
		}
		assert.NoError(t, err, tc.unit)
		assert.InDelta(t, tc.want, got, 0.001, tc.unit)
	}
}