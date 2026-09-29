package core

import (
	"testing"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util/region"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/currency"
)

func TestLoadpointsNilSlots(t *testing.T) {
	site := &Site{loadpoints: []*Loadpoint{new(Loadpoint), nil, new(Loadpoint)}}

	lps := site.Loadpoints()
	assert.Len(t, lps, 3, "disabled loadpoints must keep their slot")

	// disabled slot must be untyped nil, not a typed-nil interface
	assert.True(t, lps[1] == nil)
	assert.NotNil(t, lps[0])
	assert.NotNil(t, lps[2])

	assert.Len(t, site.activeLoadpoints(), 2)
	assert.True(t, site.IsConfigured())
}

func TestSetCountryDerivesCurrency(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	site := &Site{tariffs: &tariff.Tariffs{Currency: currency.EUR}}

	site.SetCountry("US")
	assert.Equal(t, "US", site.GetCountry())
	assert.Equal(t, currency.USD, site.tariffs.Currency, "currency should follow the country's legal tender")
}

// TestSetCountryKeepsExplicitCurrency guards against overwriting a currency
// that was configured explicitly, be it via yaml or the UI: both are
// reflected by CurrencyExplicit, unlike settings.Exists(keys.Currency) which
// only sees UI/db settings and would miss a yaml-only configuration.
func TestSetCountryKeepsExplicitCurrency(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	site := &Site{tariffs: &tariff.Tariffs{Currency: currency.EUR, CurrencyExplicit: true}}

	site.SetCountry("US")
	assert.Equal(t, currency.EUR, site.tariffs.Currency, "explicitly configured currency must not be overridden")
}

// TestSetCountryResetsDerivedCurrency guards against a previously
// country-derived currency lingering after the country is cleared or set to
// one with no single legal tender: it must fall back to the same default
// configureTariffs would use, not keep the stale derived value.
func TestSetCountryResetsDerivedCurrency(t *testing.T) {
	for _, country := range []string{"", "XX"} {
		t.Run(country, func(t *testing.T) {
			require.NoError(t, db.NewInstance("sqlite", ":memory:"))

			site := &Site{tariffs: &tariff.Tariffs{Currency: currency.USD}}

			site.SetCountry(country)
			assert.Equal(t, region.DefaultCurrency, site.tariffs.Currency)
		})
	}
}
