package gigya

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseTFARequired(t *testing.T) {
	tests := []struct {
		name string
		res  Response
		want bool
	}{
		{
			name: "known code",
			res:  Response{ErrorCode: errAccountPendingTFA},
			want: true,
		},
		{
			name: "message",
			res:  Response{ErrorMessage: "Account Pending TFA Verification"},
			want: true,
		},
		{
			name: "other error",
			res:  Response{ErrorMessage: "invalid login"},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.res.TFARequired())
		})
	}
}

func TestEmailRecordsUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		body string
		want EmailRecords
	}{
		{
			name: "array",
			body: `[{"id":"id1","plain":"user@example.com"}]`,
			want: EmailRecords{{ID: "id1", Plain: "user@example.com"}},
		},
		{
			name: "object",
			body: `{"id1":{"plain":"user@example.com"}}`,
			want: EmailRecords{{ID: "id1", Plain: "user@example.com"}},
		},
		{
			name: "single",
			body: `{"id":"id1","plain":"user@example.com"}`,
			want: EmailRecords{{ID: "id1", Plain: "user@example.com"}},
		},
		{
			name: "nested array",
			body: `{"verified":[{"id":"id1","plain":"user@example.com"}]}`,
			want: EmailRecords{{ID: "id1", Plain: "user@example.com"}},
		},
		{
			name: "keyed nested array",
			body: `{"id1":[{"plain":"user@example.com"}]}`,
			want: EmailRecords{{ID: "id1", Plain: "user@example.com"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got EmailRecords
			require.NoError(t, json.Unmarshal([]byte(tc.body), &got))
			assert.Equal(t, tc.want, got)
		})
	}
}
