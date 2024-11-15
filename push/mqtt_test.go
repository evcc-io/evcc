package push

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrepareJsonMessage(t *testing.T) {
	type test struct {
		name           string
		title          string
		message        string
		expectedResult string
		expectJson     bool
	}

	tests := []test{
		{
			name:           "plain string message",
			title:          "",
			message:        "Hello, World!",
			expectedResult: "Hello, World!",
			expectJson:     false,
		},
		{
			name:           "arbitrary non-JSON string, without title",
			title:          "Alert", // will be ignored
			message:        "Something went wrong.",
			expectedResult: "Something went wrong.",
			expectJson:     false,
		},
		{
			name:           "valid JSON message",
			title:          "",
			message:        `{"key":"value"}`,
			expectedResult: `{"key":"value"}`,
			expectJson:     true,
		},
		{
			name:           "valid JSON message with title",
			title:          "Important",
			message:        `{"key":"value"}`,
			expectedResult: `{"key":"value","title":"Important"}`,
			expectJson:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := prepareJsonMessage(tt.title, tt.message)
			if err == nil {
				var got, expected map[string]interface{}
				if tt.expectJson {
					if err := json.Unmarshal(result, &got); err != nil {
						t.Fatalf("failed to unmarshal expectedResult: %v", err)
					}
					if err := json.Unmarshal([]byte(tt.expectedResult), &expected); err != nil {
						t.Fatalf("failed to unmarshal expected JSON: %v", err)
					}
					if !equalMaps(got, expected) {
						t.Errorf("expected: %v, got: %v", expected, got)
					}
				} else {
					assert.Equal(t, tt.expectedResult, string(result))
				}
			}
		})
	}
}

func equalMaps(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if vb, found := b[k]; !found || v != vb {
			return false
		}
	}
	return true
}
