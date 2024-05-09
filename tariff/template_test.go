package tariff

import (
	"testing"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
)

var acceptable = []string{
	"missing token",         // amber, tibber
	"invalid zipcode",       // grünstromindex
	"invalid apikey format", // octopusenergy
	"missing region",        // octopusenergy
}

func TestTemplates(t *testing.T) {
	templates.TestClass(t, templates.Tariff, func(t *testing.T, values map[string]any) {
		t.Helper()

		if _, err := NewFromConfig("template", values); err != nil && !test.Acceptable(err, acceptable) {
			t.Log(values)
			t.Error(err)
		}
	})
}
