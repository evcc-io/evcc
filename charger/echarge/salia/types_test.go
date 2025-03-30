package salia

import (
	"encoding/json"
	"testing"
)

func TestAuthorizationRequestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonInput   string
		expected    AuthorizationRequest
		shouldError bool
	}{
		{
			name:      "Array direkt",
			jsonInput: `["ISO14443","9af18400"]`,
			expected:  AuthorizationRequest{Protocol: "ISO14443", Key: "9af18400"},
		},
		{
			name: "String mit Array",
			// Dieser JSON-String repräsentiert einen String, dessen Inhalt ein JSON-Array ist.
			jsonInput: `"[\"ISO14443\",\"9af18400\"]"`,
			expected:  AuthorizationRequest{Protocol: "ISO14443", Key: "9af18400"},
		},
		{
			name:        "Falsche Array-Länge",
			jsonInput:   `["ISO14443"]`,
			shouldError: true,
		},
		{
			name:        "Ungültiges JSON",
			jsonInput:   `invalid`,
			shouldError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var ar AuthorizationRequest
			err := json.Unmarshal([]byte(tc.jsonInput), &ar)
			if tc.shouldError {
				if err == nil {
					t.Errorf("Erwarteter Fehler, aber keiner erhalten")
				}
				return
			}
			if err != nil {
				t.Errorf("Unerwarteter Fehler: %v", err)
				return
			}
			if ar.Protocol != tc.expected.Protocol || ar.Key != tc.expected.Key {
				t.Errorf("Falsches Ergebnis: got %+v, expected %+v", ar, tc.expected)
			}
		})
	}
}
