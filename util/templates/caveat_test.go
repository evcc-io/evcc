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
    link: https://example.com/template
products:
  - brand: Plain
  - brand: Extra
    caveats:
      - description:
          generic: product caveat
        link: https://example.com/product
params:
  - name: host
    required: true
render: |
  type: demo
`))
	require.NoError(t, err)

	templateCaveat := Caveat{Description: TextLanguage{Generic: "template caveat"}, Link: "https://example.com/template"}

	// template caveats appended to product caveats (product first)
	require.Equal(t, []Caveat{templateCaveat}, tmpl.Products[0].Caveats)
	require.Equal(t, []Caveat{
		{Description: TextLanguage{Generic: "product caveat"}, Link: "https://example.com/product"},
		templateCaveat,
	}, tmpl.Products[1].Caveats)
}
