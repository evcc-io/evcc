package salia

import (
	"encoding/json"
	"testing"
)

func TestAuthorizationRequestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		expected  AuthorizationRequest
	}{
		{
			name: "Leerer JSON-String",
			// Dieser Input repräsentiert ein leeres Stringliteral.
			jsonInput: "\"\"",
			expected:  AuthorizationRequest{},
		},
		{
			name: "Korrekte Authentifizierungsanfrage (als JSON-String)",
			// Realer Input: ein JSON-String, der ein Array repräsentiert.
			// Das Ergebnis soll durch Unquote den Inhalt ["ISO14443","9af18400"] liefern.
			jsonInput: "\"[\\\"ISO14443\\\",\\\"9af18400\\\"]\"",
			expected:  AuthorizationRequest{Protocol: "ISO14443", Key: "9af18400"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var ar AuthorizationRequest
			err := json.Unmarshal([]byte(tc.jsonInput), &ar)
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
