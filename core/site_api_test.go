package core

import (
	"testing"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/tariff"
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

func TestSetCountryKeepsExplicitCurrency(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	settings.SetString(keys.Currency, "CHF")

	site := &Site{tariffs: &tariff.Tariffs{Currency: currency.EUR}}

	site.SetCountry("US")
	assert.Equal(t, currency.EUR, site.tariffs.Currency, "explicitly configured currency must not be overridden")
}

func TestSetCountryUnknownLeavesCurrencyUnchanged(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	site := &Site{tariffs: &tariff.Tariffs{Currency: currency.EUR}}

	site.SetCountry("XX")
	assert.Equal(t, currency.EUR, site.tariffs.Currency)
}
