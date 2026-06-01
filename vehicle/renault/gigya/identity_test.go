package gigya

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginTFARequired(t *testing.T) {
	tests := []struct {
		name string
		res  Response
		want bool
	}{
		{
			name: "account pending code",
			res:  Response{ErrorCode: errAccountPendingTFA},
			want: true,
		},
		{
			name: "TFA pending code",
			res:  Response{ErrorCode: errTFAPending},
			want: true,
		},
		{
			name: "message",
			res:  Response{ErrorMessage: "Account Pending TFA Verification"},
			want: true,
		},
		{
			name: "details",
			res:  Response{ErrorMessage: "login blocked", ErrorDetails: "TFA verification required"},
			want: true,
		},
		{
			name: "other error",
			res:  Response{ErrorCode: 403042, ErrorMessage: "invalid login", ErrorDetails: "invalid credentials"},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, LoginTFARequired(tc.res))
		})
	}
}

func TestResponseError(t *testing.T) {
	tests := []struct {
		name string
		res  Response
		want string
	}{
		{
			name: "message and details",
			res:  Response{ErrorMessage: "login failed", ErrorDetails: "invalid credentials"},
			want: "login failed: invalid credentials",
		},
		{
			name: "message",
			res:  Response{ErrorMessage: "login failed"},
			want: "login failed",
		},
		{
			name: "code",
			res:  Response{ErrorCode: 403042},
			want: "gigya error 403042",
		},
		{
			name: "empty",
			res:  Response{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.res.Error()
			if tc.want == "" {
				require.NoError(t, err)
				return
			}

			require.EqualError(t, err, tc.want)
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
