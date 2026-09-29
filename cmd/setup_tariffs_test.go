package cmd

import (
	"testing"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/currency"
)

func TestConfigureTariffsCurrency(t *testing.T) {
	tests := []struct {
		name     string
		country  string
		currency string
		want     currency.Unit
	}{
		{"defaults to EUR without country or currency", "", "", currency.EUR},
		{"derives currency from country", "US", "", currency.USD},
		{"explicit currency wins over country", "US", "CHF", currency.CHF},
		{"unknown country falls back to default", "XX", "", currency.EUR},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, db.NewInstance("sqlite", ":memory:"))
			require.NoError(t, db.Instance.AutoMigrate(&config.Config{}))

			if tc.country != "" {
				settings.SetString(keys.Country, tc.country)
			}
			if tc.currency != "" {
				settings.SetString(keys.Currency, tc.currency)
			}

			conf := &globalconfig.Tariffs{}
			tariffs, err := configureTariffs(conf)
			require.NoError(t, err)

			require.Equal(t, tc.want, tariffs.Currency)
		})
	}
}
