package gigya

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
