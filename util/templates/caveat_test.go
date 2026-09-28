package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCaveatPushdown(t *testing.T) {
	tmpl, err := fromBytes([]byte(`
template: caveats-demo
caveats:
  - description:
      generic: template caveat
products:
  - brand: Plain
  - brand: Extra
    caveats:
      - description:
          generic: product caveat
params:
  - name: host
    required: true
render: |
  type: demo
`))
	require.NoError(t, err)

	// template caveats appended to product caveats (product first)
	require.Equal(t, []Caveat{{Description: TextLanguage{Generic: "template caveat"}}}, tmpl.Products[0].Caveats)
	require.Equal(t, []Caveat{
		{Description: TextLanguage{Generic: "product caveat"}},
		{Description: TextLanguage{Generic: "template caveat"}},
	}, tmpl.Products[1].Caveats)
}

func TestCaveatDuplicate(t *testing.T) {
	// product repeats a template-level caveat
	_, err := fromBytes([]byte(`
template: caveats-dup
caveats:
  - description:
      generic: duplicate caveat
products:
  - brand: Dup
    caveats:
      - description:
          generic: duplicate caveat
params:
  - name: host
    required: true
render: |
  type: demo
`))
	require.ErrorContains(t, err, "duplicate caveat")
}
