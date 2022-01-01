package vehicle

import (
	"testing"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
)

var acceptable = []string{
	"invalid plugin type: ...",
	"missing mqtt broker configuration",
	"received status code 404 (INVALID PARAMS)", // Nissan
	"missing personID",
	"401 Unauthorized",
	"unexpected length",
	"i/o timeout",
	"no such host",
	"network is unreachable",
	"Missing required parameter", // Renault
	"error connecting: Network Error",
	"unexpected status: 401",
	"could not obtain token", // Porsche
	"missing credentials",    // Tesla
	"invalid vehicle type: hyundai",
	"invalid vehicle type: kia",
	"missing user, password or serial", // Niu
	"missing credentials id",           // Tronity
	"missing access and/or refresh token, use `evcc token` to create", // Tesla
	"login failed: Unauthorized: Authentication Failed",               // Nissan
	"login failed: no auth code",                                      // Porsche
	"invalid_client:Client authentication failed (e.g., login failure, unknown client, no client authentication included or unsupported authentication method)",   // BMW, Mini
	"login failed: oauth2: cannot fetch token: 400 Bad Request Response: {\"error\":\"invalid_request\",\"error_description\":\"Missing parameter, 'username'\"}", // Opel, DS, Citroen, PSA
}

func TestVehicleTemplates(t *testing.T) {
	test.SkipCI(t)

	for _, tmpl := range templates.ByClass(templates.Vehicle) {
		tmpl := tmpl

		// set default values for all params
		values := tmpl.Defaults(true)

		// set the template value which is needed for rendering
		values["template"] = tmpl.Template

		t.Run(tmpl.Template, func(t *testing.T) {
			t.Parallel()

			b, values, err := tmpl.RenderResult(true, values)
			if err != nil {
				t.Logf("%s: %s", tmpl.Template, b)
				t.Error(err)
			}

			_, err = NewFromConfig("template", values)
			if err != nil && !test.Acceptable(err, acceptable) {
				t.Logf("%s", tmpl.Template)
				t.Error(err)
			}
		})
	}
}
