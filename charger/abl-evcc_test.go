package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/stretchr/testify/assert"
)

func TestABLevccCapabilities(t *testing.T) {
	wb := &ABLevcc{Caps: implement.New()}

	assert.True(t, api.HasCap[api.Charger](wb))
	assert.True(t, api.HasCap[api.ChargerEx](wb))
	assert.True(t, api.HasCap[api.CurrentGetter](wb))
	assert.True(t, api.HasCap[api.Diagnosis](wb))
	assert.True(t, api.HasCap[api.Resurrector](wb))

	// only registered when the module reports its default current
	assert.False(t, api.HasCap[api.CurrentLimiter](wb))

	implement.Has(wb, implement.CurrentLimiter(wb.getMinMaxCurrent))
	assert.True(t, api.HasCap[api.CurrentLimiter](wb))
}

func TestABLevccPwm(t *testing.T) {
	for _, tc := range []struct {
		current float64
		pwm     int
	}{
		{0, ablEvccPwmMin},   // clamped
		{6, 100},             // 10.0%
		{10, 167},            // 16.7%
		{16, 267},            // 26.7%
		{32, 533},            // 53.3%
		{51, 850},            // 85.0%, upper end of the linear range
		{52, 850},            // gap between both ranges
		{53, 852},            // 85.2%
		{63, 892},            // 89.2%
		{80, 960},            // 96.0%
		{82.5, 970},          // 97.0%
		{100, ablEvccPwmMax}, // clamped
	} {
		assert.Equal(t, tc.pwm, ablEvccPwm(tc.current), "%.1fA", tc.current)
	}
}

func TestABLevccCurrent(t *testing.T) {
	for _, tc := range []struct {
		pwm     int
		current float64
	}{
		{100, 6},
		{267, 16.02},
		{850, 51},
		{852, 53},
		{960, 80},
		{970, 82.5},
	} {
		assert.InDelta(t, tc.current, ablEvccCurrent(tc.pwm), 0.001, "%d", tc.pwm)
	}
}

// every representable current must survive the round trip
func TestABLevccCurrentRoundtrip(t *testing.T) {
	for _, current := range []float64{6, 8, 10, 13, 16, 20, 24, 32, 40, 51, 53, 63, 70, 80} {
		res := ablEvccCurrent(ablEvccPwm(current))
		assert.InDelta(t, current, res, 0.03, "%.1fA", current)
	}
}

func TestABLevccStatus(t *testing.T) {
	// documented states must map to a charge status
	for code, expected := range map[int]api.ChargeStatus{
		0:  api.StatusA,
		4:  api.StatusB,
		5:  api.StatusC,
		6:  api.StatusC,
		9:  api.StatusB,
		13: api.StatusB,
		17: api.StatusA,
	} {
		assert.Equal(t, expected, ablEvccStatus[code], "%04d", code)
	}

	// error and manual states must not be mapped to a charge status
	for _, code := range []int{33, 35, 37, 39, 255} {
		_, ok := ablEvccStatus[code]
		assert.False(t, ok, "%04d", code)
	}
}
