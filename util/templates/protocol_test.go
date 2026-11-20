package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtocolsYaml(t *testing.T) {
	tmpl, err := fromBytes([]byte(`
template: protocols-none
protocols: []
`))
	require.NoError(t, err)
	assert.Empty(t, tmpl.Protocols)

	_, err = fromBytes([]byte(`
template: protocols-missing
`))
	require.ErrorContains(t, err, "missing protocols")

	_, err = fromBytes([]byte(`
template: protocols-missing-deprecated
deprecated: true
`))
	require.NoError(t, err)

	_, err = fromBytes([]byte(`
template: protocols-duplicate
protocols: [http, http]
`))
	require.ErrorContains(t, err, "duplicate protocol")

	_, err = fromBytes([]byte(`
template: protocols-invalid
protocols: [speedwire]
`))
	require.Error(t, err)
}
