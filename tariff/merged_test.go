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
