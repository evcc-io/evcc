package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// JSON numbers arrive as float64. Large integers must not be rendered in
// exponential notation, which would no longer parse as an integer.
func TestFormatValue(t *testing.T) {
	for _, tc := range []struct {
		in   any
		want string
	}{
		{float64(3), "3"},
		{float64(8899), "8899"},
		{float64(3493601102), "3493601102"}, // ten digit device serial
		{float64(1.5), "1.5"},
		{int(42), "42"},
		{"text", "text"},
	} {
		assert.Equal(t, tc.want, formatValue(tc.in))
	}
}

// end-to-end: a required int param supplied as a JSON number must satisfy the
// required check, e.g. a ten digit device serial
func TestRequiredLargeNumber(t *testing.T) {
	tmpl := &Template{
		Params: []Param{{Name: "serial", Type: TypeInt, Required: true}},
	}

	_, _, err := tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"serial": float64(3493601102),
	})
	assert.NoError(t, err, "large serial supplied as JSON number")
}
