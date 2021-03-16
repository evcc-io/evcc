package vehicle

import (
	"testing"

	"github.com/mark-sch/evcc/util/test"
)

func TestVehicles(t *testing.T) {
	acceptable := []string{
		"invalid plugin type: ...",
		"missing mqtt broker configuration",
		"received status code 404 (INVALID PARAMS)", // Nissan
		"missing personID",
		"401 Unauthorized",
		"unexpected length",
		"i/o timeout",
		"Missing required parameter", // Renault
		"error connecting: Network Error",
		"unexpected status: 401",
		"could not obtain token", // Porsche
		"missing credentials",    // Tesla
		"invalid vehicle type: hyundai",
		"invalid vehicle type: kia",
	}

	for _, tmpl := range test.ConfigTemplates("vehicle") {
		t.Run(tmpl.Name, func(t *testing.T) {
			_, err := NewFromConfig(tmpl.Type, tmpl.Config)
			if err != nil && !test.Acceptable(err, acceptable) {
				t.Logf("%s: %+v", tmpl.Name, tmpl.Config)
				t.Error(err)
			}
		})
	}
}
