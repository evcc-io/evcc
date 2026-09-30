package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinkPushdown(t *testing.T) {
	tmpl, err := fromBytes([]byte(`
template: link-demo
link: https://example.com/generic
products:
  - brand: Plain
  - brand: Own
    link: https://example.com/own
params:
  - name: host
    required: true
render: |
  type: demo
`))
	require.NoError(t, err)

	require.Equal(t, "https://example.com/generic", tmpl.Products[0].Link)
	require.Equal(t, "https://example.com/own", tmpl.Products[1].Link)
}

func TestLinkInvalid(t *testing.T) {
	_, err := fromBytes([]byte(`
template: link-invalid
link: example.com
products:
  - brand: Plain
params:
  - name: host
    required: true
render: |
  type: demo
`))
	require.ErrorContains(t, err, "invalid link")
}
