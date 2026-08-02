package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapabilityPushdown(t *testing.T) {
	tmpl, err := fromBytes([]byte(`
template: caps-demo
capabilities: ["1p3p"]
products:
  - brand: Plain
  - brand: Extra
    capabilities: ["meter"]
params:
  - name: host
    required: true
render: |
  type: demo
`))
	require.NoError(t, err)

	// template caps appended to product caps (product first)
	require.Equal(t, []Capability{Capability1p3p}, tmpl.Products[0].Capabilities)
	require.Equal(t, []Capability{CapabilityMeter, Capability1p3p}, tmpl.Products[1].Capabilities)
}

func TestCapabilityDuplicate(t *testing.T) {
	// product repeats a template-level capability
	_, err := fromBytes([]byte(`
template: caps-dup
capabilities: ["1p3p"]
products:
  - brand: Dup
    capabilities: ["1p3p"]
params:
  - name: host
    required: true
render: |
  type: demo
`))
	require.ErrorContains(t, err, "duplicate capability '1p3p'")
}
