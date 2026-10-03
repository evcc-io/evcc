package charger

import (
	"errors"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeatingStatus(t *testing.T) {
	for _, tc := range []struct {
		name   string
		power  float64
		status api.ChargeStatus
	}{
		{"idle", 0, api.StatusB},
		{"standby", heatingStandbyPower, api.StatusB},
		{"running", heatingStandbyPower + 1, api.StatusC},
	} {
		status, err := heatingStatus(func() (float64, error) { return tc.power, nil }, heatingStandbyPower)
		require.NoError(t, err, tc.name)
		assert.Equal(t, tc.status, status, tc.name)
	}

	_, err := heatingStatus(func() (float64, error) { return 0, errors.New("boom") }, heatingStandbyPower)
	assert.Error(t, err)
}
