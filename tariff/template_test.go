package tariff

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

var acceptable = []string{
	api.ErrMissingCredentials.Error(),
	api.ErrMissingToken.Error(),
	"invalid zipcode",                                  // grünstromindex
	"invalid apikey format",                            // octopusenergy
	"missing region",                                   // octopusenergy
	"missing securitytoken",                            // entsoe
	"cannot define region and postcode simultaneously", // ngeso
}

func TestEnergyPriceForecastFeatures(t *testing.T) {
	tmpl, err := templates.ByName(templates.Tariff, "energypriceforecast")
	require.NoError(t, err)

	values := tmpl.Defaults(templates.RenderModeUnitTest)
	values["average"] = true

	b, _, err := tmpl.RenderResult(templates.Tariff, templates.RenderModeUnitTest, values)
	require.NoError(t, err)

	var config struct {
		Features []string `yaml:"features"`
	}
	require.NoError(t, yaml.Unmarshal(b, &config))
	require.ElementsMatch(t, []string{"cacheable", "average"}, config.Features)
}

func TestTemplates(t *testing.T) {
	templates.TestClass(t, templates.Tariff, func(t *testing.T, values map[string]any) {
		t.Helper()

		if _, err := NewFromConfig(t.Context(), "template", values); err != nil && !test.Acceptable(err, acceptable) {
			t.Log(values)
			t.Error(err)
		}
	})
}
