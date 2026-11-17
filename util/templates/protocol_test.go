package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveProtocols(t *testing.T) {
	tc := []struct {
		name     string
		tmpl     Template
		expected []Protocol
		err      bool
	}{
		{"explicit", Template{Protocols: []Protocol{ProtocolWebSocket, ProtocolHTTP}}, []Protocol{ProtocolWebSocket, ProtocolHTTP}, false},
		{"explicit none", Template{Protocols: []Protocol{}}, []Protocol{}, false},
		{"explicit wins over modbus param", Template{Protocols: []Protocol{ProtocolOther}, Params: []Param{{Name: ParamModbus}}}, []Protocol{ProtocolOther}, false},
		{"duplicate", Template{Protocols: []Protocol{ProtocolHTTP, ProtocolHTTP}}, nil, true},
		{"modbus param", Template{Params: []Param{{Name: "usage"}, {Name: ParamModbus}}}, []Protocol{ProtocolModbus}, false},
		{"ocpp preset", Template{Params: []Param{{Preset: "ocpp"}}}, []Protocol{ProtocolOCPP}, false},
		{"eebus preset", Template{Params: []Param{{Preset: "eebus"}}}, []Protocol{ProtocolEEBus}, false},
		{"mqtt preset", Template{Params: []Param{{Preset: "mqtt"}}}, []Protocol{ProtocolMQTT}, false},
		{"other preset", Template{Params: []Param{{Preset: "vehicle-base"}}}, nil, true},
		{"missing", Template{}, nil, true},
		{"missing deprecated", Template{Deprecated: true}, nil, false},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.tmpl.ResolveProtocols()
			if tc.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, tc.tmpl.Protocols)
		})
	}
}

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
template: protocols-invalid
protocols: [speedwire]
`))
	require.Error(t, err)
}
