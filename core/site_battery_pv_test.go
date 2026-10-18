package core

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPVStartThresholdValidation(t *testing.T) {
	s := NewSite()
	require.Equal(t, 500.0, s.GetBatteryPVStartPower())
	for _, value := range []float64{math.NaN(), math.Inf(1), -1, 0, 20001} {
		require.Error(t, s.SetBatteryPVStartPower(value))
	}
	require.Equal(t, 500.0, s.GetBatteryPVStartPower())
}
