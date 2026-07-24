package vehicle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStatusMap(t *testing.T) {
	tests := []struct {
		name      string
		statusMap map[string]string
		wantErr   string
	}{
		{"nil map", nil, ""},
		{"empty map", map[string]string{}, ""},
		{"valid map", map[string]string{"charging_stopped": "B", "instant_charging": "C"}, ""},
		{"invalid value", map[string]string{"charging_stopped": "D"}, `"charging_stopped" has invalid value "D"`},
		{"empty key", map[string]string{"": "B"}, "contains an empty key"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateStatusMap(tc.statusMap)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.statusMap, got)
		})
	}
}
