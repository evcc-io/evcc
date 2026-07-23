package vehicle

import (
	"slices"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
)

var acceptable = []string{
	api.ErrMissingCredentials.Error(),
	api.ErrMissingToken.Error(),
	"missing client id",
	"invalid plugin source: ...",
	"missing mqtt broker configuration",
	"received status code 404 (INVALID PARAMS)", // Nissan
	"missing personID",
	"401 Unauthorized",
	"unexpected length",
	"i/o timeout",
	"no such host",
	"network is unreachable",
	"error connecting: Network Error",
	"unexpected status: 401",
	"discussions/17501",                            // Tesla
	"login failed: code not found",                 // Polestar
	"login failed",                                 // drivesomethinggreater (EU Data Act, eager login)
	"empty instance type- check for missing usage", // Mercedes
	"connect: connection refused",                  // MQTT
}

func TestTemplates(t *testing.T) {
	templates.TestClass(t, templates.Vehicle, func(t *testing.T, values map[string]any) {
		t.Helper()

		if _, err := NewFromConfig(t.Context(), "template", values); err != nil && !test.Acceptable(err, acceptable) {
			t.Log(values)
			t.Error(err)
		}
	})
}

// onlineVehicleFeatures render via the shared vehicle-features include, so a
// stored config may carry them; dropping the param breaks reload (discussion #31291).
var onlineVehicleFeatures = []string{"streaming", "coarsecurrent", "welcomecharge", "climaterdisabled", "autodetectdisabled", "wakeupdisabled"}

// requiredFeatureParams lists features these templates once offered as user
// params. Stored configs carry the key, so undeclaring it makes evcc fail to
// boot on the next restart (#31962). Deprecate such params, never remove them.
var requiredFeatureParams = map[string][]string{
	"vw":                    {"streaming", "coarsecurrent", "welcomecharge"},
	"audi":                  {"streaming", "coarsecurrent", "welcomecharge"},
	"seat":                  {"streaming", "coarsecurrent", "welcomecharge"},
	"cupra":                 {"streaming", "coarsecurrent", "welcomecharge"},
	"drivesomethinggreater": {"streaming", "coarsecurrent", "welcomecharge"},
}

func TestVehicleFeatureParamsConsistent(t *testing.T) {
	for _, tmpl := range templates.ByClass(templates.Vehicle, templates.WithDeprecated()) {
		if !strings.Contains(tmpl.Render, "vehicle-features") {
			continue
		}

		for _, feat := range onlineVehicleFeatures {
			// not every template ever offered every feature (e.g. wakeupdisabled is new)
			if i, _ := tmpl.ParamByName(feat); i < 0 {
				if slices.Contains(requiredFeatureParams[tmpl.Template], feat) {
					t.Errorf("%s: feature %q must stay a declared param", tmpl.Template, feat)
				}
				continue
			}

			values := tmpl.Defaults(templates.RenderModeUnitTest)
			values["template"] = tmpl.Template
			values[feat] = true

			if _, _, err := tmpl.RenderResult(templates.RenderModeInstance, values); err != nil {
				t.Errorf("%s: feature %q must stay a declared param: %v", tmpl.Template, feat, err)
			}
		}
	}
}
