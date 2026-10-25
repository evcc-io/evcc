package tariff

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTariff struct {
	rates api.Rates
	err   error
	typ   api.TariffType
}

func (m *mockTariff) Rates() (api.Rates, error) {
	return m.rates, m.err
}

func (m *mockTariff) Type() api.TariffType {
	return m.typ
}

func TestMergedRates(t *testing.T) {
	now := clock.NewMock().Now()

	primaryRates := api.Rates{
		{Start: now, End: now.Add(time.Hour), Value: 0.10},
		{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Value: 0.12},
	}

	secondaryRates := api.Rates{
		{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Value: 0.20},     // overlaps with primary
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.22}, // after primary
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.24}, // after primary
	}

	ext := &Merged{
		log:       util.NewLogger("foo"),
		primary:   &mockTariff{rates: primaryRates, typ: api.TariffTypePriceForecast},
		secondary: &mockTariff{rates: secondaryRates, typ: api.TariffTypePriceForecast},
	}

	rates, err := ext.Rates()
	require.NoError(t, err)

	// Should have primary rates plus secondary rates that start at or after primary ends
	expected := api.Rates{
		{Start: now, End: now.Add(time.Hour), Value: 0.10},
		{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Value: 0.12},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.22},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.24},
	}

	assert.Equal(t, expected, rates)
}

func TestMergedPrimaryFailure(t *testing.T) {
	now := clock.NewMock().Now()

	secondaryRates := api.Rates{
		{Start: now, End: now.Add(time.Hour), Value: 0.20},
		{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Value: 0.22},
	}

	ext := &Merged{
		log:       util.NewLogger("foo"),
		primary:   &mockTariff{err: assert.AnError, typ: api.TariffTypePriceForecast},
		secondary: &mockTariff{rates: secondaryRates, typ: api.TariffTypePriceForecast},
	}

	rates, err := ext.Rates()
	require.NoError(t, err)
	assert.Equal(t, secondaryRates, rates)
}

func TestMergedSecondaryFailure(t *testing.T) {
	now := clock.NewMock().Now()

	primaryRates := api.Rates{
		{Start: now, End: now.Add(time.Hour), Value: 0.10},
		{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Value: 0.12},
	}

	ext := &Merged{
		log:       util.NewLogger("foo"),
		primary:   &mockTariff{rates: primaryRates, typ: api.TariffTypePriceForecast},
		secondary: &mockTariff{err: assert.AnError, typ: api.TariffTypePriceForecast},
	}

	rates, err := ext.Rates()
	require.NoError(t, err)
	assert.Equal(t, primaryRates, rates)
}

func TestMergedEmptyPrimary(t *testing.T) {
	now := clock.NewMock().Now()

	secondaryRates := api.Rates{
		{Start: now, End: now.Add(time.Hour), Value: 0.20},
		{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Value: 0.22},
	}

	ext := &Merged{
		log:       util.NewLogger("foo"),
		primary:   &mockTariff{rates: api.Rates{}, typ: api.TariffTypePriceForecast},
		secondary: &mockTariff{rates: secondaryRates, typ: api.TariffTypePriceForecast},
	}

	rates, err := ext.Rates()
	require.NoError(t, err)

	// With empty primary, all secondary rates should be included
	assert.Equal(t, secondaryRates, rates)
}

func TestMergedType(t *testing.T) {
	ext := &Merged{
		log:       util.NewLogger("foo"),
		primary:   &mockTariff{typ: api.TariffTypePriceForecast},
		secondary: &mockTariff{typ: api.TariffTypePriceForecast},
	}

	assert.Equal(t, api.TariffTypePriceForecast, ext.Type())
}

func TestMergedBothFail(t *testing.T) {
	ext := &Merged{
		log:       util.NewLogger("foo"),
		primary:   &mockTariff{err: assert.AnError, typ: api.TariffTypePriceForecast},
		secondary: &mockTariff{err: assert.AnError, typ: api.TariffTypePriceForecast},
	}

	rates, err := ext.Rates()
	require.Error(t, err)
	assert.Nil(t, rates)
}

func TestMergedGapBetweenPrimaryAndSecondary(t *testing.T) {
	now := clock.NewMock().Now()

	// Primary ends at hour 2
	primaryRates := api.Rates{
		{Start: now, End: now.Add(time.Hour), Value: 0.10},
		{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Value: 0.12},
	}

	// Secondary starts at hour 4, leaving a gap from hour 2 to hour 4
	secondaryRates := api.Rates{
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.22},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.24},
	}

	ext := &Merged{
		log:       util.NewLogger("foo"),
		primary:   &mockTariff{rates: primaryRates, typ: api.TariffTypePriceForecast},
		secondary: &mockTariff{rates: secondaryRates, typ: api.TariffTypePriceForecast},
	}

	rates, err := ext.Rates()
	require.NoError(t, err)

	// Secondary should be ignored since it would create a gap
	assert.Equal(t, primaryRates, rates)
}

func TestMergedTolerance(t *testing.T) {
	now := clock.NewMock().Now()

	// 8 slots primary, secondary overlaps all of them and adds 4 more
	rates := func(n int, offset float64) api.Rates {
		var res api.Rates
		for i := range n {
			start := now.Add(time.Duration(i) * SlotDuration)
			res = append(res, api.Rate{Start: start, End: start.Add(SlotDuration), Value: 0.30 + float64(i)*0.01 + offset})
		}
		return res
	}

	for _, tc := range []struct {
		name     string
		offset   float64
		overlap  int
		expected int
	}{
		{"matching", 0.005, 8, 12},
		{"deviating", 0.05, 8, 8},
		{"too little overlap", 0, mergedMinOverlap - 1, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			secondary := rates(12, tc.offset)[8-tc.overlap:]

			m, err := NewMerged(
				&mockTariff{rates: rates(8, 0), typ: api.TariffTypePriceForecast},
				&mockTariff{rates: secondary, typ: api.TariffTypePriceForecast},
				0.01,
			)
			require.NoError(t, err)

			res, err := m.Rates()
			require.NoError(t, err)
			assert.Len(t, res, tc.expected)
			assert.Equal(t, rates(8, 0), res[:8], "primary rates are kept")
			assert.Equal(t, rates(12, tc.offset)[8:tc.expected], res[8:], "secondary rates are appended")
		})
	}
}

func TestMergedToleranceWithoutPrimary(t *testing.T) {
	secondary := &mockTariff{rates: api.Rates{{Start: time.Now(), End: time.Now().Add(SlotDuration), Value: 0.3}}, typ: api.TariffTypePriceForecast}

	m, err := NewMerged(&mockTariff{typ: api.TariffTypePriceForecast}, secondary, 0.01)
	require.NoError(t, err)

	res, err := m.Rates()
	require.NoError(t, err)
	assert.Empty(t, res, "unverified secondary rates are not used without primary")

	m, err = NewMerged(&mockTariff{err: assert.AnError, typ: api.TariffTypePriceForecast}, secondary, 0.01)
	require.NoError(t, err)

	_, err = m.Rates()
	assert.ErrorIs(t, err, assert.AnError, "primary error is not replaced by secondary rates")
}

func TestNewMergedIncompatibleTypes(t *testing.T) {
	_, err := NewMerged(
		&mockTariff{typ: api.TariffTypePriceForecast},
		&mockTariff{typ: api.TariffTypeCo2},
		0.01,
	)
	require.Error(t, err)
}
