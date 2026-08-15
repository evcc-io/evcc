package tariff

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestGridFeesDE renders the grid fee template for each grid operator
func TestGridFeesDE(t *testing.T) {
	tmpl, err := templates.ByName(templates.Tariff, "grid-fees-de")
	require.NoError(t, err)

	_, param := tmpl.ParamByName("gridoperator")
	require.NotEmpty(t, param.Choice)

	for _, op := range param.Choice {
		t.Run(op, func(t *testing.T) {
			values := tmpl.Defaults(templates.RenderModeUnitTest)
			values["template"] = tmpl.Template
			values["gridoperator"] = op

			tf, err := NewFromConfig(t.Context(), "template", values)
			require.NoError(t, err)

			rates, err := tf.Rates()
			require.NoError(t, err)
			require.NotEmpty(t, rates)

			for _, r := range rates {
				assert.Positive(t, r.Value, "%v", r)
			}
		})
	}
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
