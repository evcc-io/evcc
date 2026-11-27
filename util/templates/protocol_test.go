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

func TestProtocolsYamlOptional(t *testing.T) {
	// templates added via --template are exempt from the protocols requirement
	tmpl, err := fromBytes([]byte(`
template: protocols-optional
`), withOptionalProtocols())
	require.NoError(t, err)
	assert.Nil(t, tmpl.Protocols)

	_, err = fromBytes([]byte(`
template: protocols-duplicate-optional
protocols: [http, http]
`), withOptionalProtocols())
	require.ErrorContains(t, err, "duplicate protocol")
}
