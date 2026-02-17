package messenger

import (
	"testing"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
)

var acceptable = []string{
	// api.ErrMissingCredentials.Error(),
	// api.ErrMissingToken.Error(),
}

func TestTemplates(t *testing.T) {
	templates.TestClass(t, templates.Messenger, func(t *testing.T, values map[string]any) {
		t.Helper()

		if _, err := NewFromConfig(t.Context(), "template", values); err != nil && !test.Acceptable(err, acceptable) {
			t.Log(values)
			t.Error(err)
		}
	})
}
