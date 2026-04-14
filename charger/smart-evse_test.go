package charger

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestSmartEvseConnectedUnmarshal(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		expected bool
	}{
		{name: "bool true", value: "true", expected: true},
		{name: "bool false", value: "false", expected: false},
		{name: "int one", value: "1", expected: true},
		{name: "int zero", value: "0", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var res smartEvseRestSettings
			input := fmt.Sprintf(`{"evse":{"connected":%s}}`, tc.value)

			if err := json.Unmarshal([]byte(input), &res); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if got := bool(res.Evse.Connected); got != tc.expected {
				t.Fatalf("expected %t, got %t", tc.expected, got)
			}
		})
	}
}
